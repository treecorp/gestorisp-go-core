package bloqueio

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"gestor/internal/entity"
	"gestor/internal/helpers"
	"gestor/internal/infra/fuso"
	"gestor/internal/infra/logger"
	"gestor/internal/repositorio"
)

// loggerTag é a tag utilizada nos logs do serviço de bloqueio.
const loggerTag = "service_bloqueio"

// ProcessarFatura avalia uma fatura vencida e decide se deve bloquear o
// cliente. Retorna um ClienteBloqueado se o bloqueio foi aplicado, ou nil
// se o cliente não deve ser bloqueado.
//
// O fluxo de decisão segue a ordem:
// 1. Verifica permitir_bloqueio do contrato
// 2. Verifica se o contrato está ativo
// 3. Verifica dias_bloqueio específico do contrato (ou usa o global)
// 4. Verifica desbloqueio por confiança
// 5. Calcula dias de atraso e compara com a tolerância
func ProcessarFatura(
	repos *Repositorios,
	tag string,
	db *sql.DB,
	f entity.Fatura,
	diasBloqueioGlobal int,
) (*ClienteBloqueado, error) {
	if tag == "" {
		tag = loggerTag
	}

	// 1. Buscar contrato pelo ID
	contrato, err := repos.Contrato.BuscarContratoPorID(db, f.ContratoID)
	if err != nil {
		return nil, fmt.Errorf("buscar contrato %d: %w", f.ContratoID, err)
	}
	if contrato == nil {
		return nil, nil
	}

	// 2. Verificar permitir_bloqueio
	permitirBloqueio, err := lerPermitirBloqueio(db, f.ContratoID)
	if err != nil {
		return nil, fmt.Errorf("ler permitir_bloqueio contrato %d: %w", f.ContratoID, err)
	}
	if permitirBloqueio == 0 {
		logger.Info(tag, "Contrato %d: bloqueio nao permitido (permitir_bloqueio=0), pulando", contrato.ID)
		return nil, nil
	}

	// 3. Verificar status do contrato
	if contrato.Status != "Ativo" {
		return nil, nil
	}

	// 4. Resolver dias_bloqueio (contrato específico ou global)
	diasBloqueio, err := resolverDiasBloqueio(db, f.ContratoID, diasBloqueioGlobal)
	if err != nil {
		return nil, fmt.Errorf("resolver dias_bloqueio contrato %d: %w", f.ContratoID, err)
	}
	if diasBloqueio == -1 {
		// dias_bloqueio = -1 significa que o contrato nunca deve ser bloqueado
		return nil, nil
	}

	// 5. Verificar desbloqueio por confiança
	desbloc, err := repos.Bloqueio.LerDesbloqueioConfianca(db, f.ContratoID)
	if err != nil {
		return nil, fmt.Errorf("ler desbloqueio confianca contrato %d: %w", f.ContratoID, err)
	}

	// 6. Decidir se deve bloquear
	if !DeveBloquear(f, contrato, desbloc, diasBloqueio) {
		return nil, nil
	}

	logger.Info(tag, "Contrato %d: bloqueando (pppoe=%s, atraso=%d dias, tolerancia=%d)",
		contrato.ID, contrato.PPPoEUser, f.CalcularDiasAtraso(), diasBloqueio)

	// 7. Iniciar transação e aplicar bloqueio
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("iniciar transacao: %w", err)
	}
	defer tx.Rollback()

	if err := AplicarBloqueio(tx, contrato, f); err != nil {
		return nil, fmt.Errorf("aplicar bloqueio contrato %d: %w", contrato.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commitar transacao bloqueio %d: %w", contrato.ID, err)
	}

	logger.Sucesso(tag, "Contrato %d bloqueado com sucesso (fatura %d)", contrato.ID, f.ID)

	return &ClienteBloqueado{
		ContratoID:  contrato.ID,
		PPPoEUser:   contrato.PPPoEUser,
		PopIPv4:     contrato.PopIPv4,
		PopPort:     contrato.PopPort,
		PopUser:     contrato.PopUser,
		PopPass:     contrato.PopPass,
		ClienteNome: contrato.ClienteNome,
	}, nil
}

// DeveBloquear decide se o cliente deve ser bloqueado com base nas regras
// de negócio. Considera:
//   - Status do contrato (apenas "Ativo" pode ser bloqueado)
//   - Desbloqueio por confiança ativo (com dias restantes positivos)
//   - Dias de atraso da fatura em relação à tolerância configurada
//
// Esta é uma função pura: não realiza operações de E/S nem efeitos
// colaterais, dependendo apenas dos parâmetros recebidos.
func DeveBloquear(f entity.Fatura, contrato *entity.Contrato, desbloc *repositorio.DesbloqueioConfianca, diasBloqueio int) bool {
	// Contrato deve estar ativo
	if contrato.Status != "Ativo" {
		return false
	}

	// Desbloqueio por confiança ativo: não bloquear se ainda há dias restantes
	if desbloc != nil && desbloc.Ativo && desbloc.Dias > 0 {
		return false
	}

	// Calcular dias de atraso
	diasAtraso := f.CalcularDiasAtraso()
	if diasAtraso == 0 {
		return false
	}

	// Se atraso <= tolerância, ainda não deve bloquear
	if diasAtraso <= diasBloqueio {
		return false
	}

	return true
}

// AplicarBloqueio executa o bloqueio no banco de dados dentro de uma
// transação já iniciada. Realiza as seguintes operações:
//  1. Desativa desbloqueio por confiança (se houver registro ativo)
//  2. Insere radreply com Framed-Pool = pgcorte (se não existir)
//  3. Atualiza status do contrato para "Bloqueado"
//  4. Insere protocolo de bloqueio
//  5. Atualiza conexão para Offline
//  6. Insere log de desconexão
//
// A transação NÃO é comitada por esta função — o caller é responsável
// por chamar tx.Commit() ou tx.Rollback().
func AplicarBloqueio(tx *sql.Tx, contrato *entity.Contrato, fatura entity.Fatura) error {
	agora := fuso.Agora()
	agoraStr := agora.Format("2006-01-02 15:04:05")
	dataStr := agora.Format("2006-01-02")
	horaStr := agora.Format("15:04:05")

	// 1. Desativar desbloqueio por confiança se houver
	if err := desativarDesbloqueioConfianca(tx, fatura.ContratoID); err != nil {
		return fmt.Errorf("desativar desbloqueio confianca: %w", err)
	}

	// 2. Inserir radreply (Framed-Pool = pgcorte) se não existir
	if err := inserirRadReplyCorte(tx, contrato, fatura); err != nil {
		return fmt.Errorf("inserir radreply corte: %w", err)
	}

	// 3. Atualizar status do contrato para Bloqueado
	_, err := tx.Exec(`
		UPDATE sgp_clientes_contratos
		SET status = 'Bloqueado', data_hora_bloqueio = ?
		WHERE id = ?`, agoraStr, contrato.ID)
	if err != nil {
		return fmt.Errorf("atualizar status contrato: %w", err)
	}

	// 4. Inserir protocolo de bloqueio
	if err := inserirProtocoloBloqueio(tx, contrato, fatura, agora, agoraStr); err != nil {
		return fmt.Errorf("inserir protocolo bloqueio: %w", err)
	}

	// 5. Atualizar conexão para Offline
	if err := atualizarConexaoOffline(tx, contrato, dataStr, horaStr, agoraStr); err != nil {
		return fmt.Errorf("atualizar conexao offline: %w", err)
	}

	// 6. Inserir log de desconexão
	_, err = tx.Exec(`
		INSERT INTO sgp_clientes_logs (token, tipo, contrato_id, contrato_token, data_hora, descricao)
		VALUES (?, 'DESCONEXAO', ?, ?, ?, ?)`,
		helpers.GerarToken(), contrato.ID, contrato.Token, agoraStr,
		fmt.Sprintf("DESCONEXAO REALIZADA %s", agora.Format("02/01/2006 15:04:05")))
	if err != nil {
		return fmt.Errorf("inserir log desconexao: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Funções auxiliares internas
// ---------------------------------------------------------------------------

// lerPermitirBloqueio lê o campo permitir_bloqueio da tabela
// sgp_clientes_contratos. Retorna 0 se o bloqueio não for permitido,
// ou 1 (padrão) se for permitido.
func lerPermitirBloqueio(db *sql.DB, contratoID int) (int, error) {
	var valor sql.NullInt64
	err := db.QueryRow(`
		SELECT COALESCE(permitir_bloqueio, 1)
		FROM sgp_clientes_contratos WHERE id = ?`, contratoID).Scan(&valor)
	if err != nil {
		return 0, fmt.Errorf("consultar permitir_bloqueio: %w", err)
	}
	return int(valor.Int64), nil
}

// resolverDiasBloqueio determina quantos dias de atraso são tolerados
// antes de bloquear o contrato. Se o contrato possuir um valor específico
// (diferente de 0 ou NULL), ele é usado; caso contrário, usa o valor global.
// Retorna -1 se o contrato estiver configurado para nunca bloquear.
func resolverDiasBloqueio(db *sql.DB, contratoID int, global int) (int, error) {
	var diasStr sql.NullString
	err := db.QueryRow(`
		SELECT dias_bloqueio
		FROM sgp_clientes_contratos WHERE id = ?`, contratoID).Scan(&diasStr)
	if err != nil {
		return 0, fmt.Errorf("consultar dias_bloqueio: %w", err)
	}

	if !diasStr.Valid {
		return global, nil
	}

	trimmed := strings.TrimSpace(diasStr.String)
	if trimmed == "" {
		return global, nil
	}

	val, err := strconv.Atoi(trimmed)
	if err != nil {
		return global, nil
	}

	if val == -1 {
		return -1, nil
	}

	if val == 0 {
		return global, nil
	}

	return val, nil
}

// desativarDesbloqueioConfianca marca como 'Inativo' o registro de
// desbloqueio por confiança ativo do contrato, se houver.
func desativarDesbloqueioConfianca(tx *sql.Tx, contratoID int) error {
	_, err := tx.Exec(`
		UPDATE sgp_clientes_contratos_desbloqueio_confianca
		SET status = 'Inativo'
		WHERE contrato_id = ? AND status = 'Ativo'`, contratoID)
	if err != nil {
		return fmt.Errorf("desativar desbloqueio confianca: %w", err)
	}
	return nil
}

// inserirRadReplyCorte verifica se já existe um radreply com
// Framed-Pool = pgcorte para o usuário PPPoE. Se não existir, insere o
// registro para bloquear o acesso do cliente.
func inserirRadReplyCorte(tx *sql.Tx, contrato *entity.Contrato, fatura entity.Fatura) error {
	var id int
	err := tx.QueryRow(`
		SELECT id FROM radreply
		WHERE attribute = 'Framed-Pool' AND value = 'pgcorte' AND username = ?
		LIMIT 1`, contrato.PPPoEUser).Scan(&id)

	if err == sql.ErrNoRows {
		_, err = tx.Exec(`
			INSERT INTO radreply (username, attribute, op, value, sgp_cliente_token, sgp_contrato_token, sgp_contrato_id)
			VALUES (?, 'Framed-Pool', '=', 'pgcorte', ?, ?, ?)`,
			contrato.PPPoEUser, fatura.ClienteToken, fatura.ContratoToken, fatura.ContratoID)
		if err != nil {
			return fmt.Errorf("inserir radreply: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("verificar radreply: %w", err)
	}

	return nil
}

// inserirProtocoloBloqueio insere um registro na tabela
// sgp_clientes_contratos_protocolos documentando o bloqueio.
func inserirProtocoloBloqueio(tx *sql.Tx, contrato *entity.Contrato, fatura entity.Fatura, agora time.Time, agoraStr string) error {
	protocolo := 900000 + rand.Intn(100000)

	dadosAntigosBytes, errMarshal := json.Marshal(map[string]interface{}{
		"fatura": map[string]interface{}{
			"id":             fatura.ID,
			"contrato_id":    fatura.ContratoID,
			"contrato_token": fatura.ContratoToken,
			"cliente_token":  fatura.ClienteToken,
			"pppoe_user":     contrato.PPPoEUser,
			"vencimento":     fatura.DataVencimento,
		},
		"contrato": map[string]interface{}{
			"id":         contrato.ID,
			"token":      contrato.Token,
			"status":     contrato.Status,
			"pppoe_user": contrato.PPPoEUser,
		},
	})
	dadosAntigos := "{}"
	if errMarshal == nil {
		dadosAntigos = string(dadosAntigosBytes)
	}

	descricao := fmt.Sprintf("Bloqueio por atraso de pagamento ref fatura nº %d realizado as %s",
		fatura.ID, agora.Format("02/01/2006 15:04"))

	_, err := tx.Exec(`
		INSERT INTO sgp_clientes_contratos_protocolos
		(token, contrato_id, contrato_token, protocolo, data_hora, descricao,
		 titulo, dados_antigos, user_id, user_nome)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 'Robot')`,
		helpers.GerarToken(), contrato.ID, contrato.Token, protocolo, agoraStr,
		descricao, "Bloqueio de servico", dadosAntigos)
	if err != nil {
		return fmt.Errorf("inserir protocolo: %w", err)
	}

	return nil
}

// atualizarConexaoOffline atualiza os dados de conexão do contrato para
// Offline, incluindo data/hora de desconexão e última atividade.
func atualizarConexaoOffline(tx *sql.Tx, contrato *entity.Contrato, dataStr, horaStr, agoraStr string) error {
	// Buscar token da conexão ativa
	var conexaoToken string
	err := tx.QueryRow(`
		SELECT token FROM sgp_clientes_contratos_conexoes
		WHERE contrato_token = ? ORDER BY id DESC LIMIT 1`,
		contrato.Token).Scan(&conexaoToken)

	if err == nil {
		_, err = tx.Exec(`
			UPDATE sgp_clientes_contratos_conexoes
			SET data_desconexao = ?, hora_desconexao = ?, status = 'Offline'
			WHERE token = ?`, dataStr, horaStr, conexaoToken)
		if err != nil {
			return fmt.Errorf("atualizar conexao: %w", err)
		}
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("buscar conexao: %w", err)
	}

	// Atualizar campos de última conexão no contrato
	_, err = tx.Exec(`
		UPDATE sgp_clientes_contratos
		SET conexao = 'Offline',
		    data_ultima_conexao_atividade = ?,
		    hora_ultima_conexao_atividade = ?,
		    data_hora_ultima_conexao_atividade = ?
		WHERE id = ?`, dataStr, horaStr, agoraStr, contrato.ID)
	if err != nil {
		return fmt.Errorf("atualizar ultima conexao: %w", err)
	}

	return nil
}
