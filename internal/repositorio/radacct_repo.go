package repositorio

import (
	"database/sql"
	"fmt"
	"strings"
)

// Colunas IPv6 detectadas dinamicamente via INFORMATION_SCHEMA.
var colunasIPv6 = []string{
	"framedipv6pool",
	"framedipv6prefix",
	"delegatedipv6prefix",
	"mikrotikrealm",
}

// Colunas obrigatorias da tabela radacct usadas nas queries de SELECT/INSERT.
var colunasObrigatoriasRadacct = []string{
	"radacctid", "acctsessionid", "acctuniqueid", "username",
	"realm", "nasipaddress", "nasportid", "nasporttype",
	"acctstarttime", "acctupdatetime", "acctstoptime",
	"acctinterval", "acctsessiontime", "acctauthentic",
	"connectinfo_start", "connectinfo_stop",
	"acctinputoctets", "acctoutputoctets",
	"calledstationid", "callingstationid",
	"acctterminatecause", "servicetype", "framedprotocol",
	"framedipaddress", "groupname", "contrato_id", "contrato_pop_id",
}

// ---------------------------------------------------------------------------
// Tipos exportados
// ---------------------------------------------------------------------------

// SessaoOrphan representa uma sessao radacct sem vinculo com contrato.
type SessaoOrphan struct {
	AcctUniqueID   string
	ContratoID     int
	ContratoStatus string
	ContratoToken  string
	ContratoPopID  int
}

// ContratoResumo representa dados resumidos de um contrato.
type ContratoResumo struct {
	ID                int
	Token             string
	Status            string
	Conexao           string
	AuthType          sql.NullString
	AcctUniqueID      sql.NullString
	WSUpdateSequencia int
}

// SessaoTravada representa uma sessao radacct sem atualizacao recente.
type SessaoTravada struct {
	AcctUpdateTime string
	RadAcctID      int
	AcctUniqueID   string
	ContratoPopID  sql.NullInt64
	Username       string
}

// ContratoComSessao representa um contrato com dados da sessao radacct.
type ContratoComSessao struct {
	ID                int
	Token             string
	Status            string
	Conexao           string
	AuthType          sql.NullString
	AcctUniqueID      sql.NullString
	WSUpdateSequencia int
	RadAcctUniqueID   sql.NullString
	RadAcctStartTime  sql.NullString
	RadAcctUpdateTime sql.NullString
	RadAcctStopTime   sql.NullString
}

// PopInfo representa dados basicos de um POP usados nas operacoes radacct.
type PopInfo struct {
	ID      int
	IPv4    string
	APIPort string
	User    string
	Pass    string
	Status  string
}

// RadacctRecord representa um registro completo da tabela radacct.
type RadacctRecord struct {
	RadAcctID           int64
	AcctSessionID       string
	AcctUniqueID        string
	Username            string
	Realm               sql.NullString
	NASIPAddress        string
	NASPortID           sql.NullString
	NASPortType         sql.NullString
	AcctStartTime       sql.NullTime
	AcctUpdateTime      sql.NullTime
	AcctStopTime        sql.NullTime
	AcctInterval        sql.NullInt64
	AcctSessionTime     sql.NullInt64
	AcctAuthentic       string
	ConnectInfoStart    sql.NullString
	ConnectInfoStop     sql.NullString
	AcctInputOctets     sql.NullInt64
	AcctOutputOctets    sql.NullInt64
	CalledStationID     string
	CallingStationID    string
	AcctTerminateCause  sql.NullString
	ServiceType         sql.NullString
	FramedProtocol      sql.NullString
	FramedIPAddress     sql.NullString
	GroupName           string
	ContratoID          sql.NullInt64
	ContratoPopID       sql.NullInt64
	FramedIPv6Pool      sql.NullString
	FramedIPv6Prefix    sql.NullString
	DelegatedIPv6Prefix sql.NullString
	MikrotikRealm       sql.NullString
}

// ParColunaValor associa um nome de coluna ao seu valor para queries
// dinamicas (INSERT/UPDATE).
type ParColunaValor struct {
	Coluna string
	Valor  interface{}
}

// ---------------------------------------------------------------------------
// Funcoes auxiliares
// ---------------------------------------------------------------------------

// ExtrairData extrai apenas a parte da data (YYYY-MM-DD) de uma string
// datetime no formato "YYYY-MM-DD HH:MM:SS". Se a string for menor que
// 10 caracteres, retorna a string original.
func ExtrairData(dt string) string {
	if len(dt) >= 10 {
		return dt[:10]
	}
	return dt
}

// ExtrairHora extrai apenas a parte da hora (HH:MM:SS) de uma string
// datetime no formato "YYYY-MM-DD HH:MM:SS". Se a string for menor que
// 19 caracteres, retorna a string original.
func ExtrairHora(dt string) string {
	if len(dt) >= 19 {
		return dt[11:19]
	}
	return dt
}

// ---------------------------------------------------------------------------
// Contagem de conexoes
// ---------------------------------------------------------------------------

// ContarConexoes retorna o total de sessoes online (sem acctstoptime) e
// offline (com acctstoptime) na tabela radacct.
func ContarConexoes(db *sql.DB) (on, off int, err error) {
	query := `SELECT
		SUM(CASE WHEN acctstoptime IS NULL THEN 1 ELSE 0 END) AS online,
		SUM(CASE WHEN acctstoptime IS NOT NULL THEN 1 ELSE 0 END) AS offline
	FROM radacct`

	err = db.QueryRow(query).Scan(&on, &off)
	if err != nil {
		return 0, 0, fmt.Errorf("contar conexoes radacct: %w", err)
	}
	return on, off, nil
}

// ---------------------------------------------------------------------------
// Sessoes orphan (sem vinculo com contrato)
// ---------------------------------------------------------------------------

// BuscarSessoesOrphan retorna as sessoes radacct que ainda nao possuem
// contrato_id vinculado, limitado a 10.000 registros.
// Extraido de syncConexoesRadius (cron_1.go).
func BuscarSessoesOrphan(db *sql.DB) ([]SessaoOrphan, error) {
	rows, err := db.Query(`
		SELECT r.acctuniqueid, c.id AS contrato_id, c.status AS contrato_status,
		       c.token AS contrato_token, COALESCE(c.pop_id, 0) AS contrato_pop_id
		FROM radacct r
		INNER JOIN sgp_clientes_contratos c ON c.pppoe_user = r.username
		WHERE r.acctauthentic = 'RADIUS'
		  AND r.contrato_id IS NULL
		  AND (c.status = 'Ativo' OR c.status = 'Bloqueado')
		LIMIT 10000
	`)
	if err != nil {
		return nil, fmt.Errorf("buscar sessoes orphan: %w", err)
	}
	defer rows.Close()

	var sessoes []SessaoOrphan
	for rows.Next() {
		var s SessaoOrphan
		if err := rows.Scan(&s.AcctUniqueID, &s.ContratoID, &s.ContratoStatus, &s.ContratoToken, &s.ContratoPopID); err != nil {
			return nil, fmt.Errorf("buscar sessoes orphan: erro ao escanear: %w", err)
		}
		sessoes = append(sessoes, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("buscar sessoes orphan: erro na iteracao: %w", err)
	}

	return sessoes, nil
}

// ---------------------------------------------------------------------------
// Vinculo de sessao orphan a contrato
// ---------------------------------------------------------------------------

// VincularSessaoContrato atualiza o radacct com os IDs de contrato e POP.
func VincularSessaoContrato(db *sql.DB, acctUniqueID string, contratoID, contratoPopID int) error {
	_, err := db.Exec("UPDATE radacct SET contrato_id = ?, contrato_pop_id = ? WHERE acctuniqueid = ?",
		contratoID, contratoPopID, acctUniqueID)
	if err != nil {
		return fmt.Errorf("vincular sessao contrato: %w", err)
	}
	return nil
}

// BuscarWSUpdateSequencia retorna o ws_update_sequencia do contrato.
func BuscarWSUpdateSequencia(db *sql.DB, contratoID int) (int, error) {
	var wsSeq int
	err := db.QueryRow("SELECT COALESCE(ws_update_sequencia, 0) FROM sgp_clientes_contratos WHERE id = ?", contratoID).Scan(&wsSeq)
	if err != nil {
		return 0, fmt.Errorf("buscar ws_update_sequencia contrato %d: %w", contratoID, err)
	}
	return wsSeq, nil
}

// AtualizarContratoSessao atualiza o contrato com o acctuniqueid e
// incrementa o ws_update_sequencia.
func AtualizarContratoSessao(db *sql.DB, acctUniqueID string, wsSeq, contratoID int) error {
	_, err := db.Exec("UPDATE sgp_clientes_contratos SET acctuniqueid = ?, ws_update_sequencia = ? WHERE id = ?",
		acctUniqueID, wsSeq, contratoID)
	if err != nil {
		return fmt.Errorf("atualizar contrato sessao %d: %w", contratoID, err)
	}
	return nil
}

// BuscarSuspenderContrato retorna o campo suspender_contrato do contrato.
func BuscarSuspenderContrato(db *sql.DB, contratoID int) (string, error) {
	var suspender string
	err := db.QueryRow("SELECT COALESCE(suspender_contrato, '0') FROM sgp_clientes_contratos WHERE id = ?", contratoID).Scan(&suspender)
	if err != nil {
		return "", fmt.Errorf("buscar suspender contrato %d: %w", contratoID, err)
	}
	return suspender, nil
}

// ---------------------------------------------------------------------------
// Busca de contratos resumo
// ---------------------------------------------------------------------------

// BuscarContratosResumo retorna contratos Online com data de atividade
// anterior ao limite informado. Extraido de repararOnlineParaOffline (cron_1.go).
func BuscarContratosResumo(db *sql.DB, limite string) ([]ContratoResumo, error) {
	rows, err := db.Query(`
		SELECT id, token, status, conexao, COALESCE(auth_type, ''), acctuniqueid, COALESCE(ws_update_sequencia, 0)
		FROM sgp_clientes_contratos
		WHERE (status = 'Ativo' OR status = 'Bloqueado')
		  AND conexao = 'Online'
		  AND data_hora_ultima_conexao_atividade < ?
		ORDER BY data_hora_ultima_conexao_atividade DESC
	`, limite)
	if err != nil {
		return nil, fmt.Errorf("buscar contratos resumo: %w", err)
	}
	defer rows.Close()

	var contratos []ContratoResumo
	for rows.Next() {
		var c ContratoResumo
		if err := rows.Scan(&c.ID, &c.Token, &c.Status, &c.Conexao, &c.AuthType, &c.AcctUniqueID, &c.WSUpdateSequencia); err != nil {
			return nil, fmt.Errorf("buscar contratos resumo: erro ao escanear: %w", err)
		}
		contratos = append(contratos, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("buscar contratos resumo: erro na iteracao: %w", err)
	}

	return contratos, nil
}

// ---------------------------------------------------------------------------
// Sessoes travadas
// ---------------------------------------------------------------------------

// BuscarSessoesTravadas retorna sessoes radacct sem acctstoptime e com
// acctupdatetime anterior ao limite (10 min atras), limitado a 1000.
// Extraido de desbloquearUsuariosTravados (cron_1.go).
func BuscarSessoesTravadas(db *sql.DB, limite string) ([]SessaoTravada, error) {
	rows, err := db.Query(`
		SELECT DATE_FORMAT(acctupdatetime, '%Y-%m-%d %H:%i:%s') AS acctupdatetime,
		       radacctid, acctuniqueid, contrato_pop_id, username
		FROM radacct
		WHERE acctauthentic = 'RADIUS'
		  AND acctstoptime IS NULL
		  AND acctupdatetime < ?
		LIMIT 1000
	`, limite)
	if err != nil {
		return nil, fmt.Errorf("buscar sessoes travadas: %w", err)
	}
	defer rows.Close()

	var sessoes []SessaoTravada
	for rows.Next() {
		var s SessaoTravada
		if err := rows.Scan(&s.AcctUpdateTime, &s.RadAcctID, &s.AcctUniqueID, &s.ContratoPopID, &s.Username); err != nil {
			return nil, fmt.Errorf("buscar sessoes travadas: erro ao escanear: %w", err)
		}
		sessoes = append(sessoes, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("buscar sessoes travadas: erro na iteracao: %w", err)
	}

	return sessoes, nil
}

// FecharSessaoTravada atualiza a radacct definindo acctstoptime e
// acctterminatecause para a sessao.
func FecharSessaoTravada(db *sql.DB, acctUpdateTime, causa, acctUniqueID string) error {
	_, err := db.Exec("UPDATE radacct SET acctstoptime = ?, acctterminatecause = ? WHERE acctuniqueid = ?",
		acctUpdateTime, causa, acctUniqueID)
	if err != nil {
		return fmt.Errorf("fechar sessao travada %s: %w", acctUniqueID, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Busca de contratos com sessao
// ---------------------------------------------------------------------------

// BuscarContratosComSessao retorna contratos Ativos/Bloqueados que possuem
// acctuniqueid, com dados da sessao radacct via LEFT JOIN.
// Extraido de syncConexoesRadiusStatus (cron_1.go).
func BuscarContratosComSessao(db *sql.DB, popID int) ([]ContratoComSessao, error) {
	rows, err := db.Query(`
		SELECT c.id, c.token, c.status, c.conexao, c.auth_type,
		       c.acctuniqueid, COALESCE(c.ws_update_sequencia, 0),
		       r.acctuniqueid,
		       DATE_FORMAT(r.acctstarttime, '%Y-%m-%d %H:%i:%s') AS acctstarttime,
		       DATE_FORMAT(r.acctupdatetime, '%Y-%m-%d %H:%i:%s') AS acctupdatetime,
		       DATE_FORMAT(r.acctstoptime, '%Y-%m-%d %H:%i:%s') AS acctstoptime
		FROM sgp_clientes_contratos c
		LEFT JOIN radacct r ON r.acctuniqueid = c.acctuniqueid
		WHERE c.acctuniqueid IS NOT NULL
		  AND (c.status = 'Ativo' OR c.status = 'Bloqueado')
		ORDER BY c.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("buscar contratos com sessao: %w", err)
	}
	defer rows.Close()

	var contratos []ContratoComSessao
	for rows.Next() {
		var c ContratoComSessao
		if err := rows.Scan(&c.ID, &c.Token, &c.Status, &c.Conexao, &c.AuthType,
			&c.AcctUniqueID, &c.WSUpdateSequencia,
			&c.RadAcctUniqueID, &c.RadAcctStartTime, &c.RadAcctUpdateTime, &c.RadAcctStopTime); err != nil {
			return nil, fmt.Errorf("buscar contratos com sessao: erro ao escanear: %w", err)
		}
		contratos = append(contratos, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("buscar contratos com sessao: erro na iteracao: %w", err)
	}

	return contratos, nil
}

// ---------------------------------------------------------------------------
// Atualizacao de status de conexao no contrato
// ---------------------------------------------------------------------------

// AtualizarContratoOfflineParaOnline altera o contrato de Offline para
// Online, atualizando auth_type e timestamps.
func AtualizarContratoOfflineParaOnline(db *sql.DB, contratoID int, dataHora, data, hora string) error {
	_, err := db.Exec(`
		UPDATE sgp_clientes_contratos
		SET conexao = 'Online', auth_type = 'Radius',
		    data_hora_ultima_conexao_atividade = ?,
		    data_ultima_conexao_atividade = ?,
		    hora_ultima_conexao_atividade = ?
		WHERE id = ?
	`, dataHora, data, hora, contratoID)
	if err != nil {
		return fmt.Errorf("atualizar contrato offline->online %d: %w", contratoID, err)
	}
	return nil
}

// AtualizarContratoAtividade atualiza apenas os timestamps de atividade
// do contrato (mantendo a conexao atual).
func AtualizarContratoAtividade(db *sql.DB, contratoID int, dataHora, data, hora string) error {
	_, err := db.Exec(`
		UPDATE sgp_clientes_contratos
		SET data_hora_ultima_conexao_atividade = ?,
		    data_ultima_conexao_atividade = ?,
		    hora_ultima_conexao_atividade = ?
		WHERE id = ?
	`, dataHora, data, hora, contratoID)
	if err != nil {
		return fmt.Errorf("atualizar contrato atividade %d: %w", contratoID, err)
	}
	return nil
}

// AtualizarContratoOnlineParaOffline altera o contrato de Online para
// Offline, limpando o acctuniqueid e ajustando auth_type.
func AtualizarContratoOnlineParaOffline(db *sql.DB, contratoID int, dataHora, data, hora string) error {
	_, err := db.Exec(`
		UPDATE sgp_clientes_contratos
		SET conexao = 'Offline', auth_type = 'Radius',
		    data_hora_ultima_conexao_atividade = ?,
		    data_ultima_conexao_atividade = ?,
		    hora_ultima_conexao_atividade = ?,
		    acctuniqueid = NULL
		WHERE id = ?
	`, dataHora, data, hora, contratoID)
	if err != nil {
		return fmt.Errorf("atualizar contrato online->offline %d: %w", contratoID, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Check de conexoes locais
// ---------------------------------------------------------------------------

// BuscarStatusConexaoLocal retorna o status da ultima conexao local do
// contrato.
func BuscarStatusConexaoLocal(db *sql.DB, contratoToken string) (string, error) {
	var status string
	err := db.QueryRow(`
		SELECT COALESCE(status, 'Offline') FROM sgp_clientes_contratos_conexoes
		WHERE contrato_token = ? ORDER BY id DESC LIMIT 1
	`, contratoToken).Scan(&status)
	if err != nil {
		return "", fmt.Errorf("buscar status conexao local: %w", err)
	}
	return status, nil
}

// AtualizarConexaoLocalOffline marca a conexao local como Offline.
func AtualizarConexaoLocalOffline(db *sql.DB, contratoToken, data, hora string) error {
	_, err := db.Exec(`
		UPDATE sgp_clientes_contratos_conexoes
		SET data_desconexao = ?, hora_desconexao = ?, status = 'Offline'
		WHERE contrato_token = ?
	`, data, hora, contratoToken)
	if err != nil {
		return fmt.Errorf("atualizar conexao local offline: %w", err)
	}
	return nil
}

// AtualizarContratoLocalOffline marca o contrato como Offline com timestamps.
func AtualizarContratoLocalOffline(db *sql.DB, contratoID int, data, hora, dataHora string) error {
	_, err := db.Exec(`
		UPDATE sgp_clientes_contratos
		SET conexao = 'Offline',
		    data_ultima_conexao_atividade = ?,
		    hora_ultima_conexao_atividade = ?,
		    data_hora_ultima_conexao_atividade = ?
		WHERE id = ?
	`, data, hora, dataHora, contratoID)
	if err != nil {
		return fmt.Errorf("atualizar contrato local offline %d: %w", contratoID, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Reparo Offline -> Online
// ---------------------------------------------------------------------------

// BuscarContratosOfflineComSessao retorna contratos marcados como Offline
// que possuem sessao ativa na radacct (sem acctstoptime).
func BuscarContratosOfflineComSessao(db *sql.DB) ([]ContratoResumo, error) {
	rows, err := db.Query(`
		SELECT c.id, c.token, c.status, c.conexao, COALESCE(c.auth_type, ''),
		       COALESCE(c.ws_update_sequencia, 0),
		       r.acctuniqueid
		FROM sgp_clientes_contratos c
		INNER JOIN radacct r ON r.contrato_id = c.id AND r.acctstoptime IS NULL
		WHERE c.conexao = 'Offline' AND (c.status = 'Ativo' OR c.status = 'Bloqueado')
		ORDER BY c.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("buscar contratos offline com sessao: %w", err)
	}
	defer rows.Close()

	var contratos []ContratoResumo
	for rows.Next() {
		var c ContratoResumo
		if err := rows.Scan(&c.ID, &c.Token, &c.Status, &c.Conexao, &c.AuthType, &c.WSUpdateSequencia, &c.AcctUniqueID); err != nil {
			return nil, fmt.Errorf("buscar contratos offline com sessao: erro ao escanear: %w", err)
		}
		contratos = append(contratos, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("buscar contratos offline com sessao: erro na iteracao: %w", err)
	}

	return contratos, nil
}

// ---------------------------------------------------------------------------
// Reparo Online -> Offline
// ---------------------------------------------------------------------------

// AtualizarContratoForceOffline marca o contrato como Offline e limpa
// acctuniqueid, forçando a desconexao.
func AtualizarContratoForceOffline(db *sql.DB, contratoID int) error {
	_, err := db.Exec(`
		UPDATE sgp_clientes_contratos
		SET conexao = 'Offline', auth_type = 'Radius', acctuniqueid = NULL
		WHERE id = ?
	`, contratoID)
	if err != nil {
		return fmt.Errorf("atualizar contrato force offline %d: %w", contratoID, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Log de desconexao
// ---------------------------------------------------------------------------

// AdicionarLogDesconexao insere ou incrementa o contador de desconexao do
// contrato na data atual. Se ja existir registro na data, incrementa os
// contadores; caso contrario, insere novo registro.
// A data utilizada e CURDATE() do proprio MySQL, evitando dependencia de
// fuso horario da aplicacao.
func AdicionarLogDesconexao(db *sql.DB, contratoID int, contratoToken, token string) error {
	var tokenExistente string
	err := db.QueryRow(`
		SELECT token FROM sgp_clientes_contratos_desconexoes
		WHERE contrato_token = ? AND data = CURDATE()
		ORDER BY data DESC LIMIT 1
	`, contratoToken).Scan(&tokenExistente)
	if err == sql.ErrNoRows {
		_, err = db.Exec(`
			INSERT INTO sgp_clientes_contratos_desconexoes (token, contrato_id, contrato_token, data, contador, contador2)
			VALUES (?, ?, ?, CURDATE(), 1, 1)
		`, token, contratoID, contratoToken)
		if err != nil {
			return fmt.Errorf("adicionar log desconexao: erro ao inserir: %w", err)
		}
	} else if err == nil {
		_, err = db.Exec("UPDATE sgp_clientes_contratos_desconexoes SET contador = contador + 1, contador2 = contador2 + 1 WHERE token = ?", tokenExistente)
		if err != nil {
			return fmt.Errorf("adicionar log desconexao: erro ao atualizar: %w", err)
		}
	} else {
		return fmt.Errorf("adicionar log desconexao: erro ao consultar: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Schema introspection
// ---------------------------------------------------------------------------

// DetectarColunasRadacct consulta INFORMATION_SCHEMA para detectar as
// colunas disponiveis na tabela radacct da base atual.
func DetectarColunasRadacct(db *sql.DB) (map[string]bool, error) {
	query := `SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = (SELECT DATABASE())
		AND TABLE_NAME = 'radacct'
		ORDER BY ORDINAL_POSITION`

	linhas, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("detectar colunas radacct: %w", err)
	}
	defer linhas.Close()

	existentes := make(map[string]bool)
	for linhas.Next() {
		var nome string
		if err := linhas.Scan(&nome); err != nil {
			return nil, fmt.Errorf("detectar colunas radacct: erro ao escanear: %w", err)
		}
		existentes[nome] = true
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("detectar colunas radacct: erro na iteracao: %w", err)
	}

	return existentes, nil
}

// DetectarColunasArquivo consulta INFORMATION_SCHEMA para detectar as
// colunas disponiveis na tabela radacct_arquivo da base atual.
func DetectarColunasArquivo(db *sql.DB) (map[string]bool, error) {
	query := `SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = (SELECT DATABASE())
		AND TABLE_NAME = 'radacct_arquivo'
		ORDER BY ORDINAL_POSITION`

	linhas, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("detectar colunas arquivo: %w", err)
	}
	defer linhas.Close()

	existentes := make(map[string]bool)
	for linhas.Next() {
		var nome string
		if err := linhas.Scan(&nome); err != nil {
			return nil, fmt.Errorf("detectar colunas arquivo: erro ao escanear: %w", err)
		}
		existentes[nome] = true
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("detectar colunas arquivo: erro na iteracao: %w", err)
	}

	return existentes, nil
}

// ---------------------------------------------------------------------------
// RADIUS file sync — busca registros pendentes
// ---------------------------------------------------------------------------

// BuscarRadacctPendenteArquivo retorna registros radacct com acctstoptime
// preenchido e contrato_id vinculado, limitado a 4999 linhas, ordenados
// do mais recente para o mais antigo.
func BuscarRadacctPendenteArquivo(db *sql.DB, colunasRadacct map[string]bool) ([]RadacctRecord, error) {
	colunasSELECT := MontarListaColunasSELECT(colunasRadacct)

	query := fmt.Sprintf(`SELECT %s FROM radacct
	WHERE acctauthentic = 'RADIUS'
		AND acctstoptime IS NOT NULL
		AND contrato_id IS NOT NULL
		AND (SELECT COUNT(*) FROM radacct
			WHERE acctstoptime IS NOT NULL
			AND contrato_id IS NOT NULL) > 1
	ORDER BY radacctid DESC
	LIMIT 4999`, colunasSELECT)

	linhas, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("buscar radacct pendente arquivo: %w", err)
	}
	defer linhas.Close()

	var resultado []RadacctRecord
	for linhas.Next() {
		var r RadacctRecord
		targets := MontarScanTargets(&r, colunasRadacct)
		if err := linhas.Scan(targets...); err != nil {
			return nil, fmt.Errorf("buscar radacct pendente arquivo: erro ao escanear: %w", err)
		}
		resultado = append(resultado, r)
	}

	if len(resultado) == 0 {
		return nil, nil
	}

	return resultado, linhas.Err()
}

// MontarListaColunasSELECT monta a lista de colunas para o SELECT com
// base nas colunas detectadas, incluindo colunas obrigatorias e IPv6.
func MontarListaColunasSELECT(colunasRadacct map[string]bool) string {
	var cols []string
	for _, nome := range colunasObrigatoriasRadacct {
		if colunasRadacct[nome] {
			cols = append(cols, nome)
		}
	}
	for _, nome := range colunasIPv6 {
		if colunasRadacct[nome] {
			cols = append(cols, nome)
		}
	}
	return strings.Join(cols, ", ")
}

// MontarScanTargets retorna a lista de ponteiros para scan na ordem das
// colunas selecionadas.
func MontarScanTargets(r *RadacctRecord, colunasRadacct map[string]bool) []interface{} {
	m := map[string]interface{}{
		"radacctid":           &r.RadAcctID,
		"acctsessionid":       &r.AcctSessionID,
		"acctuniqueid":        &r.AcctUniqueID,
		"username":            &r.Username,
		"realm":               &r.Realm,
		"nasipaddress":        &r.NASIPAddress,
		"nasportid":           &r.NASPortID,
		"nasporttype":         &r.NASPortType,
		"acctstarttime":       &r.AcctStartTime,
		"acctupdatetime":      &r.AcctUpdateTime,
		"acctstoptime":        &r.AcctStopTime,
		"acctinterval":        &r.AcctInterval,
		"acctsessiontime":     &r.AcctSessionTime,
		"acctauthentic":       &r.AcctAuthentic,
		"connectinfo_start":   &r.ConnectInfoStart,
		"connectinfo_stop":    &r.ConnectInfoStop,
		"acctinputoctets":     &r.AcctInputOctets,
		"acctoutputoctets":    &r.AcctOutputOctets,
		"calledstationid":     &r.CalledStationID,
		"callingstationid":    &r.CallingStationID,
		"acctterminatecause":  &r.AcctTerminateCause,
		"servicetype":         &r.ServiceType,
		"framedprotocol":      &r.FramedProtocol,
		"framedipaddress":     &r.FramedIPAddress,
		"groupname":           &r.GroupName,
		"contrato_id":         &r.ContratoID,
		"contrato_pop_id":     &r.ContratoPopID,
		"framedipv6pool":      &r.FramedIPv6Pool,
		"framedipv6prefix":    &r.FramedIPv6Prefix,
		"delegatedipv6prefix": &r.DelegatedIPv6Prefix,
		"mikrotikrealm":       &r.MikrotikRealm,
	}

	var targets []interface{}
	for _, nome := range colunasObrigatoriasRadacct {
		if colunasRadacct[nome] {
			targets = append(targets, m[nome])
		}
	}
	for _, nome := range colunasIPv6 {
		if colunasRadacct[nome] {
			targets = append(targets, m[nome])
		}
	}
	return targets
}

// ProcessarRegistro orquestra a migracao de um registro radacct para a
// tabela radacct_arquivo dentro de uma transacao. Insere ou atualiza o
// registro na arquivo e deleta da tabela radacct original.
func ProcessarRegistro(tx *sql.Tx, rec RadacctRecord, colunasDisponiveis map[string]bool) error {
	cols, _ := MontarColunasValores(rec, colunasDisponiveis)

	err := InserirRadacct(tx, cols)
	if err != nil {
		if strings.Contains(err.Error(), "1062") {
			err = AtualizarRadacctPorAcctUniqueID(tx, cols, rec.AcctUniqueID)
		}
		if err != nil {
			return fmt.Errorf("processar registro: erro ao escrever em radacct_arquivo: %w", err)
		}
	}

	if err := DeletarRadacct(tx, rec.RadAcctID); err != nil {
		return fmt.Errorf("processar registro: erro ao deletar de radacct: %w", err)
	}

	return nil
}

// MontarColunasValores monta a lista de pares coluna-valor a partir de um
// RadacctRecord, filtrando apenas as colunas disponiveis.
func MontarColunasValores(rec RadacctRecord, colunasDisponiveis map[string]bool) ([]ParColunaValor, string) {
	todos := []ParColunaValor{
		{"radacctid", rec.RadAcctID},
		{"acctsessionid", rec.AcctSessionID},
		{"acctuniqueid", rec.AcctUniqueID},
		{"username", rec.Username},
		{"realm", rec.Realm},
		{"nasipaddress", rec.NASIPAddress},
		{"nasportid", rec.NASPortID},
		{"nasporttype", rec.NASPortType},
		{"acctstarttime", rec.AcctStartTime},
		{"acctupdatetime", rec.AcctUpdateTime},
		{"acctstoptime", rec.AcctStopTime},
		{"acctinterval", rec.AcctInterval},
		{"acctsessiontime", rec.AcctSessionTime},
		{"acctauthentic", rec.AcctAuthentic},
		{"connectinfo_start", rec.ConnectInfoStart},
		{"connectinfo_stop", rec.ConnectInfoStop},
		{"acctinputoctets", rec.AcctInputOctets},
		{"acctoutputoctets", rec.AcctOutputOctets},
		{"calledstationid", rec.CalledStationID},
		{"callingstationid", rec.CallingStationID},
		{"acctterminatecause", rec.AcctTerminateCause},
		{"servicetype", rec.ServiceType},
		{"framedprotocol", rec.FramedProtocol},
		{"framedipaddress", rec.FramedIPAddress},
		{"groupname", rec.GroupName},
		{"contrato_id", rec.ContratoID},
		{"contrato_pop_id", rec.ContratoPopID},
		{"framedipv6pool", rec.FramedIPv6Pool},
		{"framedipv6prefix", rec.FramedIPv6Prefix},
		{"delegatedipv6prefix", rec.DelegatedIPv6Prefix},
		{"mikrotikrealm", rec.MikrotikRealm},
	}

	var filtrado []ParColunaValor
	for _, p := range todos {
		if colunasDisponiveis[p.Coluna] {
			filtrado = append(filtrado, p)
		}
	}

	var colunas []string
	for _, p := range filtrado {
		colunas = append(colunas, p.Coluna)
	}

	return filtrado, strings.Join(colunas, ", ")
}

// InserirRadacct insere um novo registro na tabela radacct_arquivo com as
// colunas e valores fornecidos.
func InserirRadacct(tx *sql.Tx, cols []ParColunaValor) error {
	var nomes []string
	var placeholders []string
	var valores []interface{}

	for _, p := range cols {
		nomes = append(nomes, p.Coluna)
		placeholders = append(placeholders, "?")
		valores = append(valores, p.Valor)
	}

	query := fmt.Sprintf("INSERT INTO radacct_arquivo (%s) VALUES (%s)",
		strings.Join(nomes, ", "),
		strings.Join(placeholders, ", "))

	_, err := tx.Exec(query, valores...)
	return err
}

// AtualizarRadacct atualiza um registro existente na tabela radacct_arquivo
// pelo radacctid.
func AtualizarRadacct(tx *sql.Tx, cols []ParColunaValor, radacctid int64) error {
	var sets []string
	var valores []interface{}

	for _, p := range cols {
		if p.Coluna == "radacctid" {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = ?", p.Coluna))
		valores = append(valores, p.Valor)
	}

	valores = append(valores, radacctid)

	query := fmt.Sprintf("UPDATE radacct_arquivo SET %s WHERE radacctid = ?",
		strings.Join(sets, ", "))

	_, err := tx.Exec(query, valores...)
	return err
}

// AtualizarRadacctPorAcctUniqueID atualiza um registro na tabela
// radacct_arquivo pelo acctuniqueid (exceto as colunas acctuniqueid e
// radacctid).
func AtualizarRadacctPorAcctUniqueID(tx *sql.Tx, cols []ParColunaValor, acctUniqueID string) error {
	var sets []string
	var valores []interface{}

	for _, p := range cols {
		if p.Coluna == "acctuniqueid" || p.Coluna == "radacctid" {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = ?", p.Coluna))
		valores = append(valores, p.Valor)
	}

	valores = append(valores, acctUniqueID)

	query := fmt.Sprintf("UPDATE radacct_arquivo SET %s WHERE acctuniqueid = ?",
		strings.Join(sets, ", "))

	_, err := tx.Exec(query, valores...)
	return err
}

// DeletarRadacct deleta um registro da tabela radacct pelo radacctid.
func DeletarRadacct(tx *sql.Tx, radacctid int64) error {
	query := "DELETE FROM radacct WHERE radacctid = ?"
	_, err := tx.Exec(query, radacctid)
	return err
}
