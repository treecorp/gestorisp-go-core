package worker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"time"

	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/fuso"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/routeros"
)

type faturaVencida struct {
	ID            int
	ContratoID    int
	ContratoToken string
	ClienteToken  string
	PPPoEUser     string
	PopID         int
	Vencimento    string
}

type contratoBloqueio struct {
	ID               int
	Token            string
	Status           string
	PPPoEUser        string
	PermitirBloqueio int
	DiasBloqueio     sql.NullInt64
}

type desbloqueioConfianca struct {
	ID              int
	DataHoraBloqueio string
}

type clienteBloqueado struct {
	PPPoEUser string
	PopID     int
}

func HandlerListarClientesVencidos(instancia dominio.Instancia) error {
	tag := "listar_clientes_vencidos"

	diaSemana := fuso.Agora().Weekday()
	if diaSemana == time.Saturday || diaSemana == time.Sunday {
		logger.Info(tag, "Final de semana, ignorando bloqueio")
		return nil
	}

	db, err := banco.ConectarInstancia(
		instancia.EnvDBHost, instancia.EnvDBPort,
		instancia.EnvDBUser, instancia.EnvDBPass, instancia.EnvDBName,
	)
	if err != nil {
		return fmt.Errorf("falha ao conectar na instancia %d: %w", instancia.ID, err)
	}
	defer banco.FecharConexaoInstancia(db, tag)

	logger.Inicio(tag, "Instancia %d (%s)", instancia.ID, instancia.EnvDBName)

	diasBloqueio := lerDiasBloqueio(tag, db)

	faturas, err := buscarFaturasVencidas(tag, db)
	if err != nil {
		return fmt.Errorf("erro ao buscar faturas vencidas: %w", err)
	}
	if len(faturas) == 0 {
		logger.Info(tag, "Nenhuma fatura vencida encontrada")
		return nil
	}

	logger.Info(tag, "%d faturas vencidas para processar", len(faturas))

	var bloqueados []clienteBloqueado
	for _, f := range faturas {
		bloqueado, err := processarFatura(tag, db, f, diasBloqueio)
		if err != nil {
			logger.Erro(tag, "Erro ao processar fatura %d: %v", f.ID, err)
			continue
		}
		if bloqueado != nil {
			bloqueados = append(bloqueados, *bloqueado)
		}
	}

	if len(bloqueados) == 0 {
		logger.Info(tag, "Nenhum cliente a bloquear")
		return nil
	}

	logger.Info(tag, "%d clientes para desconectar da RouterBoard", len(bloqueados))

	pops := carregarPops(db)
	for _, cb := range bloqueados {
		desconectarCliente(tag, cb, pops)
	}

	logger.Sucesso(tag, "Bloqueio realizado com sucesso para %d contratos", len(bloqueados))
	return nil
}

func lerDiasBloqueio(tag string, db *sql.DB) int {
	var dias sql.NullInt64
	err := db.QueryRow("SELECT dias_bloqueio FROM sgp_parametros LIMIT 1").Scan(&dias)
	if err != nil || !dias.Valid {
		logger.Info(tag, "dias_bloqueio padrao: 5")
		return 5
	}
	logger.Info(tag, "dias_bloqueio global: %d", dias.Int64)
	return int(dias.Int64)
}

func buscarFaturasVencidas(tag string, db *sql.DB) ([]faturaVencida, error) {
	dataFinal := fuso.Agora().Format("2006-01-02")
	rows, err := db.Query(`
		SELECT f.id, f.contrato_id, f.contrato_token, f.cliente_token,
		       c.pppoe_user, COALESCE(c.pop_id, 0) AS pop_id,
		       DATE_FORMAT(f.vencimento, '%Y-%m-%d') AS vencimento
		FROM sgp_clientes_faturas f
		INNER JOIN sgp_clientes_contratos c ON c.id = f.contrato_id
		WHERE c.isento = 'Não'
		  AND f.status = 'Pendente'
		  AND f.vencimento BETWEEN '2018-06-01' AND ?
		  AND f.vencimento < ?
		ORDER BY f.vencimento ASC
	`, dataFinal, dataFinal)
	if err != nil {
		return nil, fmt.Errorf("erro na query de faturas: %w", err)
	}
	defer rows.Close()

	var faturas []faturaVencida
	for rows.Next() {
		var f faturaVencida
		if err := rows.Scan(&f.ID, &f.ContratoID, &f.ContratoToken,
			&f.ClienteToken, &f.PPPoEUser, &f.PopID, &f.Vencimento); err != nil {
			return nil, fmt.Errorf("erro ao scanear fatura: %w", err)
		}
		faturas = append(faturas, f)
	}
	if faturas == nil {
		return nil, nil
	}
	return faturas, nil
}

func processarFatura(tag string, db *sql.DB, f faturaVencida, diasBloqueioGlobal int) (*clienteBloqueado, error) {
	contrato, err := lerContrato(db, f.ContratoID)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler contrato %d: %w", f.ContratoID, err)
	}
	if contrato == nil {
		return nil, nil
	}

	if contrato.PermitirBloqueio == 0 {
		logger.Info(tag, "Contrato %d: bloqueio nao permitido (permitir_bloqueio=0), pulando", contrato.ID)
		return nil, nil
	}

	if contrato.Status != "Ativo" {
		return nil, nil
	}

	desbloc, err := lerDesbloqueioConfianca(db, f.ContratoID)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler desbloqueio confianca contrato %d: %w", f.ContratoID, err)
	}

	if desbloc != nil {
		hojeStr := fuso.Agora().Format("2006-01-02 00:00:00")
		if hojeStr < desbloc.DataHoraBloqueio {
			return nil, nil
		}
	}

	diasAtraso := calcularDiasAtraso(f.Vencimento)

	diasBloqueio := diasBloqueioGlobal
	if contrato.DiasBloqueio.Valid {
		diasBloqueio = int(contrato.DiasBloqueio.Int64)
	}

	if diasAtraso <= diasBloqueio {
		return nil, nil
	}

	logger.Info(tag, "Contrato %d: bloqueando (pppoe=%s, atraso=%d dias, tolerancia=%d)",
		contrato.ID, contrato.PPPoEUser, diasAtraso, diasBloqueio)

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("erro ao iniciar transacao: %w", err)
	}
	defer tx.Rollback()

	if desbloc != nil {
		_, err = tx.Exec("UPDATE sgp_clientes_contratos_desbloqueio_confianca SET status = 'Inativo' WHERE id = ?", desbloc.ID)
		if err != nil {
			return nil, fmt.Errorf("erro ao desativar trust-unblock: %w", err)
		}
	}

	var radreplyID int
	err = tx.QueryRow("SELECT id FROM radreply WHERE attribute = 'Framed-Pool' AND value = 'pgcorte' AND username = ? LIMIT 1",
		contrato.PPPoEUser).Scan(&radreplyID)
	if err == sql.ErrNoRows {
		_, err = tx.Exec(`
			INSERT INTO radreply (username, attribute, op, value, sgp_cliente_token, sgp_contrato_token, sgp_contrato_id)
			VALUES (?, 'Framed-Pool', '=', 'pgcorte', ?, ?, ?)
		`, contrato.PPPoEUser, f.ClienteToken, f.ContratoToken, f.ContratoID)
		if err != nil {
			return nil, fmt.Errorf("erro ao inserir radreply: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("erro ao verificar radreply: %w", err)
	}

	agora := fuso.Agora()
	agoraStr := agora.Format("2006-01-02 15:04:05")
	dataStr := agora.Format("2006-01-02")
	horaStr := agora.Format("15:04:05")

	_, err = tx.Exec("UPDATE sgp_clientes_contratos SET status = 'Bloqueado', data_hora_bloqueio = ? WHERE id = ?",
		agoraStr, contrato.ID)
	if err != nil {
		return nil, fmt.Errorf("erro ao atualizar status contrato: %w", err)
	}

	protocolo := 900000 + mrand.Intn(100000)
	dadosAntigosBytes, errMarshal := json.Marshal(map[string]interface{}{
		"fatura": map[string]interface{}{
			"id":             f.ID,
			"contrato_id":    f.ContratoID,
			"contrato_token": f.ContratoToken,
			"cliente_token":  f.ClienteToken,
			"pppoe_user":     f.PPPoEUser,
			"vencimento":     f.Vencimento,
		},
		"contrato": map[string]interface{}{
			"id":        contrato.ID,
			"token":     contrato.Token,
			"status":    contrato.Status,
			"pppoe_user": contrato.PPPoEUser,
		},
	})
	dadosAntigos := "{}"
	if errMarshal == nil {
		dadosAntigos = string(dadosAntigosBytes)
	}

	descricao := fmt.Sprintf("Bloqueio por atraso de pagamento ref fatura nº %d realizado as %s",
		f.ID, agora.Format("02/01/2006 15:04"))

	_, err = tx.Exec(`
		INSERT INTO sgp_clientes_contratos_protocolos
		(token, contrato_id, contrato_token, protocolo, data_hora, descricao,
		 titulo, dados_antigos, user_id, user_nome)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 'Robot')
	`, gerarToken(), contrato.ID, contrato.Token, protocolo, agoraStr,
		descricao, "Bloqueio de servico", dadosAntigos)
	if err != nil {
		return nil, fmt.Errorf("erro ao inserir protocolo: %w", err)
	}

	var conexaoToken string
	err = tx.QueryRow("SELECT token FROM sgp_clientes_contratos_conexoes WHERE contrato_token = ? ORDER BY id DESC LIMIT 1",
		contrato.Token).Scan(&conexaoToken)
	if err == nil {
		_, err = tx.Exec(`
			UPDATE sgp_clientes_contratos_conexoes
			SET data_desconexao = ?, hora_desconexao = ?, status = 'Offline'
			WHERE token = ?
		`, dataStr, horaStr, conexaoToken)
		if err != nil {
			return nil, fmt.Errorf("erro ao atualizar conexao: %w", err)
		}
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("erro ao buscar conexao: %w", err)
	}

	_, err = tx.Exec(`
		UPDATE sgp_clientes_contratos
		SET conexao = 'Offline',
		    data_ultima_conexao_atividade = ?,
		    hora_ultima_conexao_atividade = ?,
		    data_hora_ultima_conexao_atividade = ?
		WHERE id = ?
	`, dataStr, horaStr, agoraStr, contrato.ID)
	if err != nil {
		return nil, fmt.Errorf("erro ao atualizar conexao contrato: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO sgp_clientes_logs (token, tipo, contrato_id, contrato_token, data_hora, descricao)
		VALUES (?, 'DESCONEXAO', ?, ?, ?, ?)
	`, gerarToken(), contrato.ID, contrato.Token, agoraStr,
		fmt.Sprintf("DESCONEXAO REALIZADA %s", agora.Format("02/01/2006 15:04:05")))
	if err != nil {
		return nil, fmt.Errorf("erro ao inserir log desconexao: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("erro ao commitar transacao: %w", err)
	}

	adicionarLogDesconexao(tag, db, contrato.ID, contrato.Token)

	logger.Sucesso(tag, "Contrato %d bloqueado com sucesso (fatura %d)", contrato.ID, f.ID)

	return &clienteBloqueado{PPPoEUser: contrato.PPPoEUser, PopID: f.PopID}, nil
}

func lerContrato(db *sql.DB, contratoID int) (*contratoBloqueio, error) {
	var c contratoBloqueio
	err := db.QueryRow(`
		SELECT id, token, status, pppoe_user,
		       COALESCE(permitir_bloqueio, 1) AS permitir_bloqueio,
		       CAST(dias_bloqueio AS UNSIGNED) AS dias_bloqueio
		FROM sgp_clientes_contratos WHERE id = ?
	`, contratoID).Scan(&c.ID, &c.Token, &c.Status, &c.PPPoEUser,
		&c.PermitirBloqueio, &c.DiasBloqueio)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao scanear contrato: %w", err)
	}
	return &c, nil
}

func lerDesbloqueioConfianca(db *sql.DB, contratoID int) (*desbloqueioConfianca, error) {
	var d desbloqueioConfianca
	err := db.QueryRow(`
		SELECT id, DATE_FORMAT(data_hora_bloqueio, '%Y-%m-%d %H:%i:%s') AS data_hora_bloqueio
		FROM sgp_clientes_contratos_desbloqueio_confianca
		WHERE contrato_id = ? AND status = 'Ativo'
		ORDER BY id DESC LIMIT 1
	`, contratoID).Scan(&d.ID, &d.DataHoraBloqueio)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao scanear desbloqueio: %w", err)
	}
	return &d, nil
}

func calcularDiasAtraso(vencimento string) int {
	venc, err := time.Parse("2006-01-02", vencimento)
	if err != nil {
		return 0
	}
	hoje := fuso.Agora().Truncate(24 * time.Hour)
	dias := int(hoje.Sub(venc).Hours() / 24)
	if dias < 0 {
		return 0
	}
	return dias
}

func desconectarCliente(tag string, cb clienteBloqueado, pops map[int]popInfo) {
	pop, ok := pops[cb.PopID]
	if !ok {
		logger.Aviso(tag, "Cliente %s: POP %d nao encontrado", cb.PPPoEUser, cb.PopID)
		return
	}

	conn, err := routeros.Conectar(routeros.DadosConexao{
		IPv4: pop.IPv4,
		Port: pop.APIPort,
		User: pop.User,
		Pass: pop.Pass,
	})
	if err != nil {
		logger.Aviso(tag, "Cliente %s: POP %d (%s) inacessivel: %v", cb.PPPoEUser, pop.ID, pop.IPv4, err)
		return
	}
	defer conn.Close()

	ativo, sessionID, err := routeros.VerificarUsuarioAtivo(conn, cb.PPPoEUser)
	if err != nil {
		logger.Aviso(tag, "Cliente %s: erro ao consultar RB: %v", cb.PPPoEUser, err)
		return
	}

	if !ativo {
		logger.Info(tag, "Cliente %s: ja desconectado da RB", cb.PPPoEUser)
		return
	}

	if err := routeros.DesconectarUsuario(conn, sessionID); err != nil {
		logger.Aviso(tag, "Cliente %s: erro ao desconectar na RB: %v", cb.PPPoEUser, err)
		return
	}

	logger.Info(tag, "Cliente %s: desconectado da RB (POP %d)", cb.PPPoEUser, pop.ID)
}

