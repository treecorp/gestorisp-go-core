package worker

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/fuso"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/routeros"
)

func gerarToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type sessaoOrphan struct {
	AcctUniqueID   string
	ContratoID     int
	ContratoStatus string
	ContratoToken  string
	ContratoPopID  int
}

type contratoResumo struct {
	ID               int
	Token            string
	Status           string
	Conexao          string
	AuthType         sql.NullString
	AcctUniqueID     sql.NullString
	WSUpdateSequencia int
}

type sessaoRadacct struct {
	AcctUniqueID  sql.NullString
	AcctStartTime sql.NullString
	AcctUpdateTime sql.NullString
	AcctStopTime  sql.NullString
}

type sessaoTravada struct {
	AcctUpdateTime string
	RadAcctID      int
	AcctUniqueID   string
	ContratoPopID  int
	Username       string
}

type popInfo struct {
	ID     int
	IPv4   string
	APIPort string
	User   string
	Pass   string
	Status string
}

func HandlerCron1(instancia dominio.Instancia) error {
	tag := "cron_1"
	db, err := banco.ConectarInstancia(
		instancia.EnvDBHost, instancia.EnvDBPort,
		instancia.EnvDBUser, instancia.EnvDBPass, instancia.EnvDBName,
	)
	if err != nil {
		return fmt.Errorf("falha ao conectar na instancia %d: %w", instancia.ID, err)
	}
	defer banco.FecharConexaoInstancia(db, tag)

	logger.Inicio(tag, "Instancia %d (%s) - sync_1", instancia.ID, instancia.EnvDBName)

	if err := syncConexoesRadius(tag, db); err != nil {
		logger.Erro(tag, "sync_conexoes_radius: %v", err)
	}
	if err := syncConexoesRadiusStatus(tag, db); err != nil {
		logger.Erro(tag, "sync_conexoes_radius_status: %v", err)
	}
	if err := desbloquearUsuariosTravados(tag, db); err != nil {
		logger.Erro(tag, "desbloqueia_user_bloqueado: %v", err)
	}

	logger.Info(tag, "Instancia %d - reparar status", instancia.ID)

	if err := repararOfflineParaOnline(tag, db); err != nil {
		logger.Erro(tag, "reparar_offline_para_online: %v", err)
	}
	if err := repararOnlineParaOffline(tag, db); err != nil {
		logger.Erro(tag, "reparar_online_para_offline: %v", err)
	}

	logger.Sucesso(tag, "Instancia %d processada com sucesso", instancia.ID)
	return nil
}

func syncConexoesRadius(tag string, db *sql.DB) error {
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
		return fmt.Errorf("erro ao buscar sessoes orphan: %w", err)
	}
	defer rows.Close()

	var sessoes []sessaoOrphan
	for rows.Next() {
		var s sessaoOrphan
		if err := rows.Scan(&s.AcctUniqueID, &s.ContratoID, &s.ContratoStatus, &s.ContratoToken, &s.ContratoPopID); err != nil {
			return fmt.Errorf("erro ao scanear sessao orphan: %w", err)
		}
		sessoes = append(sessoes, s)
	}
	if sessoes == nil {
		return nil
	}

	logger.Info(tag, "sync_conexoes_radius: %d sessoes orphan", len(sessoes))

	for _, s := range sessoes {
		_, err := db.Exec("UPDATE radacct SET contrato_id = ?, contrato_pop_id = ? WHERE acctuniqueid = ?",
			s.ContratoID, s.ContratoPopID, s.AcctUniqueID)
		if err != nil {
			logger.Erro(tag, "erro ao atualizar radacct %s: %v", s.AcctUniqueID, err)
			continue
		}

		var wsSeq int
		err = db.QueryRow("SELECT COALESCE(ws_update_sequencia, 0) FROM sgp_clientes_contratos WHERE id = ?", s.ContratoID).Scan(&wsSeq)
		if err != nil {
			logger.Erro(tag, "erro ao ler ws_update_sequencia do contrato %d: %v", s.ContratoID, err)
			continue
		}

		_, err = db.Exec("UPDATE sgp_clientes_contratos SET acctuniqueid = ?, ws_update_sequencia = ? WHERE id = ?",
			s.AcctUniqueID, wsSeq+1, s.ContratoID)
		if err != nil {
			logger.Erro(tag, "erro ao atualizar contrato %d: %v", s.ContratoID, err)
			continue
		}

		if s.ContratoStatus == "Ativo" {
			var suspender string
			err = db.QueryRow("SELECT COALESCE(suspender_contrato, '0') FROM sgp_clientes_contratos WHERE id = ?", s.ContratoID).Scan(&suspender)
			if err != nil {
				continue
			}
			if suspender == "0" {
				adicionarLogDesconexao(tag, db, s.ContratoID, s.ContratoToken)
			}
		}
	}

	return nil
}

func adicionarLogDesconexao(tag string, db *sql.DB, contratoID int, contratoToken string) {
	hoje := fuso.Agora().Format("2006-01-02")
	amanha := fuso.Agora().Format("2006-01-02")

	var token string
	err := db.QueryRow(`
		SELECT token FROM sgp_clientes_contratos_desconexoes
		WHERE contrato_token = ? AND data BETWEEN ? AND ?
		ORDER BY data DESC LIMIT 1
	`, contratoToken, hoje, amanha).Scan(&token)
	if err == sql.ErrNoRows {
		novoToken := gerarToken()
		_, err = db.Exec(`
			INSERT INTO sgp_clientes_contratos_desconexoes (token, contrato_id, contrato_token, data, contador, contador2)
			VALUES (?, ?, ?, ?, 1, 1)
		`, novoToken, contratoID, contratoToken, hoje)
		if err != nil {
			logger.Erro(tag, "erro ao inserir log desconexao: %v", err)
		}
	} else if err == nil {
		_, err = db.Exec("UPDATE sgp_clientes_contratos_desconexoes SET contador = contador + 1, contador2 = contador2 + 1 WHERE token = ?", token)
		if err != nil {
			logger.Erro(tag, "erro ao atualizar log desconexao: %v", err)
		}
	}
}

func syncConexoesRadiusStatus(tag string, db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, token, status, conexao, auth_type, acctuniqueid
		FROM sgp_clientes_contratos
		WHERE acctuniqueid IS NOT NULL AND (status = 'Ativo' OR status = 'Bloqueado')
		ORDER BY id ASC
	`)
	if err != nil {
		return fmt.Errorf("erro ao buscar contratos: %w", err)
	}
	defer rows.Close()

	var contratos []contratoResumo
	for rows.Next() {
		var c contratoResumo
		if err := rows.Scan(&c.ID, &c.Token, &c.Status, &c.Conexao, &c.AuthType, &c.AcctUniqueID); err != nil {
			return fmt.Errorf("erro ao scanear contrato: %w", err)
		}
		contratos = append(contratos, c)
	}
	if contratos == nil {
		return nil
	}

	for _, c := range contratos {
		if !c.AuthType.Valid || c.AuthType.String == "Local" {
			checkConexoesLocal(tag, db, c)
		}

		var s sessaoRadacct
		err := db.QueryRow(`
			SELECT acctuniqueid, acctstarttime, acctupdatetime, acctstoptime
			FROM radacct WHERE acctuniqueid = ?
		`, c.AcctUniqueID.String).Scan(&s.AcctUniqueID, &s.AcctStartTime, &s.AcctUpdateTime, &s.AcctStopTime)
		if err != nil {
			continue
		}

		if (c.Status == "Ativo" || c.Status == "Bloqueado") && s.AcctUniqueID.Valid && s.AcctUniqueID.String == c.AcctUniqueID.String {
			if !s.AcctStopTime.Valid {
				agora := fuso.Agora()
				if c.Conexao == "Offline" {
					_, err = db.Exec(`
						UPDATE sgp_clientes_contratos
						SET conexao = 'Online', auth_type = 'Radius',
						    data_hora_ultima_conexao_atividade = ?,
						    data_ultima_conexao_atividade = ?,
						    hora_ultima_conexao_atividade = ?
						WHERE id = ?
					`, s.AcctStartTime.String, extrairData(s.AcctStartTime.String), extrairHora(s.AcctStartTime.String), c.ID)
				} else {
					_, err = db.Exec(`
						UPDATE sgp_clientes_contratos
						SET data_hora_ultima_conexao_atividade = ?,
						    data_ultima_conexao_atividade = ?,
						    hora_ultima_conexao_atividade = ?
						WHERE id = ?
					`, s.AcctUpdateTime.String, extrairData(s.AcctUpdateTime.String), extrairHora(s.AcctUpdateTime.String), c.ID)
				}
				_ = agora
			} else {
				_, err = db.Exec(`
					UPDATE sgp_clientes_contratos
					SET conexao = 'Offline', auth_type = 'Radius',
					    data_hora_ultima_conexao_atividade = ?,
					    data_ultima_conexao_atividade = ?,
					    hora_ultima_conexao_atividade = ?,
					    acctuniqueid = NULL
					WHERE id = ?
				`, s.AcctStopTime.String, extrairData(s.AcctStopTime.String), extrairHora(s.AcctStopTime.String), c.ID)
			}
			if err != nil {
				logger.Erro(tag, "erro atualizar status contrato %d: %v", c.ID, err)
			}
		}
	}

	return nil
}

func extrairData(dt string) string {
	if len(dt) >= 10 {
		return dt[:10]
	}
	return dt
}

func extrairHora(dt string) string {
	if len(dt) >= 19 {
		return dt[11:19]
	}
	return dt
}

func checkConexoesLocal(tag string, db *sql.DB, c contratoResumo) {
	var statusConexao string
	err := db.QueryRow(`
		SELECT COALESCE(status, 'Offline') FROM sgp_clientes_contratos_conexoes
		WHERE contrato_token = ? ORDER BY id DESC LIMIT 1
	`, c.Token).Scan(&statusConexao)
	if err != nil {
		return
	}
	if statusConexao == "Offline" {
		return
	}

	agora := fuso.Agora()
	_, err = db.Exec(`
		UPDATE sgp_clientes_contratos_conexoes
		SET data_desconexao = ?, hora_desconexao = ?, status = 'Offline'
		WHERE contrato_token = ?
	`, agora.Format("2006-01-02"), agora.Format("15:04:05"), c.Token)
	if err != nil {
		logger.Erro(tag, "erro atualizar conexoes_local contrato %d: %v", c.ID, err)
	}

	_, err = db.Exec(`
		UPDATE sgp_clientes_contratos
		SET conexao = 'Offline',
		    data_ultima_conexao_atividade = ?,
		    hora_ultima_conexao_atividade = ?,
		    data_hora_ultima_conexao_atividade = ?
		WHERE id = ?
	`, agora.Format("2006-01-02"), agora.Format("15:04:05"), agora.Format("2006-01-02 15:04:05"), c.ID)
	if err != nil {
		logger.Erro(tag, "erro atualizar contrato %d (local): %v", c.ID, err)
	}
}

func desbloquearUsuariosTravados(tag string, db *sql.DB) error {
	limite := fuso.Agora().Add(-10 * time.Minute)
	rows, err := db.Query(`
		SELECT acctupdatetime, radacctid, acctuniqueid, contrato_pop_id, username
		FROM radacct
		WHERE acctauthentic = 'RADIUS'
		  AND acctstoptime IS NULL
		  AND acctupdatetime < ?
		LIMIT 1000
	`, limite.Format("2006-01-02 15:04:05"))
	if err != nil {
		return fmt.Errorf("erro ao buscar sessoes travadas: %w", err)
	}
	defer rows.Close()

	var sessoes []sessaoTravada
	for rows.Next() {
		var s sessaoTravada
		if err := rows.Scan(&s.AcctUpdateTime, &s.RadAcctID, &s.AcctUniqueID, &s.ContratoPopID, &s.Username); err != nil {
			return fmt.Errorf("erro ao scanear sessao travada: %w", err)
		}
		sessoes = append(sessoes, s)
	}
	if sessoes == nil {
		return nil
	}

	pops := carregarPops(db)

	for _, s := range sessoes {
		pop, ok := pops[s.ContratoPopID]
		if !ok || pop.Status != "OPERACIONAL" {
			continue
		}

		conn, err := routeros.Conectar(routeros.DadosConexao{
			IPv4: pop.IPv4,
			Port: pop.APIPort,
			User: pop.User,
			Pass: pop.Pass,
		})
		if err != nil {
			logger.Aviso(tag, "POP %d: %v - preservando sessao %s", pop.ID, err, s.AcctUniqueID)
			continue
		}

		ativo, _, _ := routeros.VerificarUsuarioAtivo(conn, s.Username)
		conn.Close()

		causa := "NAS Error"
		if ativo {
			causa = "NAS Error(d)"
		}

		_, err = db.Exec("UPDATE radacct SET acctstoptime = ?, acctterminatecause = ? WHERE acctuniqueid = ?",
			s.AcctUpdateTime, causa, s.AcctUniqueID)
		if err != nil {
			logger.Erro(tag, "erro ao fechar sessao %s: %v", s.AcctUniqueID, err)
		}
	}

	return nil
}

func carregarPops(db *sql.DB) map[int]popInfo {
	rows, err := db.Query("SELECT id, ipv4, api_port, user, pass, status FROM sgp_pops")
	if err != nil {
		return nil
	}
	defer rows.Close()

	pops := make(map[int]popInfo)
	for rows.Next() {
		var p popInfo
		if err := rows.Scan(&p.ID, &p.IPv4, &p.APIPort, &p.User, &p.Pass, &p.Status); err != nil {
			continue
		}
		pops[p.ID] = p
	}
	return pops
}

func repararOfflineParaOnline(tag string, db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, token, status, conexao, COALESCE(auth_type, ''), acctuniqueid, COALESCE(ws_update_sequencia, 0)
		FROM sgp_clientes_contratos
		WHERE conexao = 'Offline' AND (status = 'Ativo' OR status = 'Bloqueado')
		ORDER BY id ASC
	`)
	if err != nil {
		return fmt.Errorf("erro ao buscar contratos offline: %w", err)
	}
	defer rows.Close()

	var contratos []contratoResumo
	for rows.Next() {
		var c contratoResumo
		if err := rows.Scan(&c.ID, &c.Token, &c.Status, &c.Conexao, &c.AuthType, &c.AcctUniqueID, &c.WSUpdateSequencia); err != nil {
			return fmt.Errorf("erro ao scanear contrato: %w", err)
		}
		contratos = append(contratos, c)
	}
	if contratos == nil {
		return nil
	}

	agora := fuso.Agora()
	for _, c := range contratos {
		var acctID string
		err := db.QueryRow(`
			SELECT acctuniqueid FROM radacct
			WHERE contrato_id = ? AND acctstoptime IS NULL
			LIMIT 1
		`, c.ID).Scan(&acctID)
		if err != nil {
			continue
		}

		_, err = db.Exec(`
			UPDATE sgp_clientes_contratos
			SET acctuniqueid = ?, ws_update_sequencia = ?,
			    conexao = 'Online', auth_type = 'Radius',
			    data_hora_ultima_conexao_atividade = ?,
			    data_ultima_conexao_atividade = ?,
			    hora_ultima_conexao_atividade = ?
			WHERE id = ?
		`, acctID, c.WSUpdateSequencia+1,
			agora.Format("2006-01-02 15:04:05"), agora.Format("2006-01-02"), agora.Format("15:04:05"),
			c.ID)
		if err != nil {
			logger.Erro(tag, "erro reparar contrato %d: %v", c.ID, err)
		}
	}

	return nil
}

func repararOnlineParaOffline(tag string, db *sql.DB) error {
	limite := fuso.Agora().Add(-30 * time.Minute)
	rows, err := db.Query(`
		SELECT id, token, status, conexao, COALESCE(auth_type, ''), acctuniqueid, COALESCE(ws_update_sequencia, 0)
		FROM sgp_clientes_contratos
		WHERE (status = 'Ativo' OR status = 'Bloqueado')
		  AND conexao = 'Online'
		  AND data_hora_ultima_conexao_atividade < ?
		ORDER BY data_hora_ultima_conexao_atividade DESC
	`, limite.Format("2006-01-02 15:04:05"))
	if err != nil {
		return fmt.Errorf("erro ao buscar contratos online: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var c contratoResumo
		if err := rows.Scan(&c.ID, &c.Token, &c.Status, &c.Conexao, &c.AuthType, &c.AcctUniqueID, &c.WSUpdateSequencia); err != nil {
			return fmt.Errorf("erro ao scanear contrato: %w", err)
		}

		_, err := db.Exec(`
			UPDATE sgp_clientes_contratos
			SET conexao = 'Offline', auth_type = 'Radius', acctuniqueid = NULL
			WHERE id = ?
		`, c.ID)
		if err != nil {
			logger.Erro(tag, "erro ao marcar offline contrato %d: %v", c.ID, err)
		}
	}

	return nil
}
