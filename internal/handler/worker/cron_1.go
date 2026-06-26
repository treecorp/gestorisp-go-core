package worker

import (
	"database/sql"
	"fmt"
	"time"

	"gestor/internal/entity"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/fuso"
	"gestor/internal/helpers"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/routeros"
	"gestor/internal/repositorio"
)

// HandlerCron1 executa o cron de sincronizacao de sessoes radius para uma
// instancia. Inclui sync de conexoes, desbloqueio de sessoes travadas e
// reparo de status Offline/Online.
func HandlerCron1(instancia entity.Instancia) error {
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

// syncConexoesRadius vincula sessoes orphan da radacct aos contratos.
func syncConexoesRadius(tag string, db *sql.DB) error {
	sessoes, err := repositorio.BuscarSessoesOrphan(db)
	if err != nil {
		return fmt.Errorf("erro ao buscar sessoes orphan: %w", err)
	}
	if sessoes == nil {
		return nil
	}

	logger.Info(tag, "sync_conexoes_radius: %d sessoes orphan encontradas", len(sessoes))

	for _, s := range sessoes {
		logger.Info(tag, "vinculando sessao %s ao contrato %d (pop %d)", s.AcctUniqueID, s.ContratoID, s.ContratoPopID)

		if err := repositorio.VincularSessaoContrato(db, s.AcctUniqueID, s.ContratoID, s.ContratoPopID); err != nil {
			logger.Erro(tag, "erro ao atualizar radacct %s: %v", s.AcctUniqueID, err)
			continue
		}

		wsSeq, err := repositorio.BuscarWSUpdateSequencia(db, s.ContratoID)
		if err != nil {
			logger.Erro(tag, "erro ao ler ws_update_sequencia do contrato %d: %v", s.ContratoID, err)
			continue
		}

		if err := repositorio.AtualizarContratoSessao(db, s.AcctUniqueID, wsSeq+1, s.ContratoID); err != nil {
			logger.Erro(tag, "erro ao atualizar contrato %d: %v", s.ContratoID, err)
			continue
		}

		logger.Info(tag, "contrato %d atualizado: acctuniqueid=%s ws_seq=%d", s.ContratoID, s.AcctUniqueID, wsSeq+1)

		if s.ContratoStatus == "Ativo" {
			suspender, err := repositorio.BuscarSuspenderContrato(db, s.ContratoID)
			if err != nil {
				continue
			}
			if suspender == "0" {
				token := gerarToken()
				_ = repositorio.AdicionarLogDesconexao(db, s.ContratoID, s.ContratoToken, token)
			}
		}
	}

	logger.Info(tag, "sync_conexoes_radius: %d sessoes vinculadas", len(sessoes))
	return nil
}

// syncConexoesRadiusStatus sincroniza o status Online/Offline dos contratos
// com base na sessao radacct ativa.
func syncConexoesRadiusStatus(tag string, db *sql.DB) error {
	contratos, err := repositorio.BuscarContratosComSessao(db, 0)
	if err != nil {
		return fmt.Errorf("erro ao buscar contratos com sessao: %w", err)
	}
	if contratos == nil {
		logger.Info(tag, "sync_conexoes_radius_status: nenhum contrato com acctuniqueid")
		return nil
	}

	logger.Info(tag, "sync_conexoes_radius_status: %d contratos para verificar", len(contratos))
	online := 0
	offline := 0

	for _, c := range contratos {
		if !c.AuthType.Valid || c.AuthType.String == "Local" {
			checkConexoesLocal(tag, db, repositorio.ContratoResumo{
				ID: c.ID, Token: c.Token, Status: c.Status,
				Conexao: c.Conexao, AuthType: c.AuthType, AcctUniqueID: c.AcctUniqueID,
			})
		}

		if (c.Status == "Ativo" || c.Status == "Bloqueado") && c.RadAcctUniqueID.Valid && c.RadAcctUniqueID.String == c.AcctUniqueID.String {
			if !c.RadAcctStopTime.Valid {
				if c.Conexao == "Offline" {
					logger.Info(tag, "contrato %d: Offline->Online (sessao ativa desde %s)", c.ID, c.RadAcctStartTime.String)
					if err := repositorio.AtualizarContratoOfflineParaOnline(db, c.ID,
						c.RadAcctStartTime.String,
						repositorio.ExtrairData(c.RadAcctStartTime.String),
						repositorio.ExtrairHora(c.RadAcctStartTime.String)); err != nil {
						logger.Erro(tag, "erro atualizar status contrato %d: %v", c.ID, err)
					}
					online++
				} else {
					if err := repositorio.AtualizarContratoAtividade(db, c.ID,
						c.RadAcctUpdateTime.String,
						repositorio.ExtrairData(c.RadAcctUpdateTime.String),
						repositorio.ExtrairHora(c.RadAcctUpdateTime.String)); err != nil {
						logger.Erro(tag, "erro atualizar atividade contrato %d: %v", c.ID, err)
					}
				}
			} else {
				logger.Info(tag, "contrato %d: Online->Offline (sessao encerrada em %s)", c.ID, c.RadAcctStopTime.String)
				if err := repositorio.AtualizarContratoOnlineParaOffline(db, c.ID,
					c.RadAcctStopTime.String,
					repositorio.ExtrairData(c.RadAcctStopTime.String),
					repositorio.ExtrairHora(c.RadAcctStopTime.String)); err != nil {
					logger.Erro(tag, "erro atualizar status contrato %d: %v", c.ID, err)
				}
				offline++
			}
		}
	}

	logger.Info(tag, "sync_conexoes_radius_status: %d Online, %d Offline", online, offline)
	return nil
}

// checkConexoesLocal verifica conexoes locais de contratos com auth_type=Local.
func checkConexoesLocal(tag string, db *sql.DB, c repositorio.ContratoResumo) {
	statusConexao, err := repositorio.BuscarStatusConexaoLocal(db, c.Token)
	if err != nil {
		return
	}
	if statusConexao == "Offline" {
		return
	}

	logger.Info(tag, "contrato %d: conexao local Online->Offline (auth_type=Local)", c.ID)

	agora := fuso.Agora()
	if err := repositorio.AtualizarConexaoLocalOffline(db, c.Token,
		agora.Format("2006-01-02"), agora.Format("15:04:05")); err != nil {
		logger.Erro(tag, "erro atualizar conexoes_local contrato %d: %v", c.ID, err)
	}

	if err := repositorio.AtualizarContratoLocalOffline(db, c.ID,
		agora.Format("2006-01-02"), agora.Format("15:04:05"), agora.Format("2006-01-02 15:04:05")); err != nil {
		logger.Erro(tag, "erro atualizar contrato %d (local): %v", c.ID, err)
	}
}

// desbloquearUsuariosTravados fecha sessoes radacct sem atualizacao recente.
func desbloquearUsuariosTravados(tag string, db *sql.DB) error {
	limite := fuso.Agora().Add(-10 * time.Minute)
	sessoes, err := repositorio.BuscarSessoesTravadas(db, limite.Format("2006-01-02 15:04:05"))
	if err != nil {
		return fmt.Errorf("erro ao buscar sessoes travadas: %w", err)
	}
	if sessoes == nil {
		logger.Info(tag, "desbloquear: nenhuma sessao travada")
		return nil
	}

	logger.Info(tag, "desbloquear: %d sessoes travadas (>10min sem update)", len(sessoes))
	pops := carregarPopsCR1(db)
	fechadas := 0
	puladas := 0

	for _, s := range sessoes {
		if !s.ContratoPopID.Valid {
			logger.Info(tag, "sessao %s: sem POP vinculado, preservando", s.AcctUniqueID)
			puladas++
			continue
		}
		pop, ok := pops[int(s.ContratoPopID.Int64)]
		if !ok || pop.Status != "OPERACIONAL" {
			logger.Info(tag, "sessao %s: POP %d nao operacional, preservando", s.AcctUniqueID, s.ContratoPopID.Int64)
			puladas++
			continue
		}

		conn, err := routeros.Conectar(routeros.DadosConexao{
			IPv4: pop.IPv4,
			Port: pop.APIPort,
			User: pop.User,
			Pass: pop.Pass,
		})
		if err != nil {
			logger.Aviso(tag, "sessao %s: POP %d (%s) inacessivel, preservando", s.AcctUniqueID, pop.ID, pop.IPv4)
			puladas++
			continue
		}

		ativo, _, err := routeros.VerificarUsuarioAtivo(conn, s.Username)
		conn.Close()

		causa := "NAS Error"
		if ativo {
			causa = "NAS Error(d)"
		}

		if err != nil {
			logger.Aviso(tag, "sessao %s: erro ao consultar RB, preservando: %v", s.AcctUniqueID, err)
			puladas++
			continue
		}

		logger.Info(tag, "sessao %s: user=%s POP=%d ativo_na_rb=%v causa=%s acctupdatetime=%s",
			s.AcctUniqueID, s.Username, pop.ID, ativo, causa, s.AcctUpdateTime)

		if err := repositorio.FecharSessaoTravada(db, s.AcctUpdateTime, causa, s.AcctUniqueID); err != nil {
			logger.Erro(tag, "erro ao fechar sessao %s: %v", s.AcctUniqueID, err)
			continue
		}
		fechadas++
	}

	logger.Info(tag, "desbloquear: %d fechadas, %d preservadas", fechadas, puladas)
	return nil
}

// carregarPopsCR1 carrega todos os POPs do banco para uso no cron.
func carregarPopsCR1(db *sql.DB) map[int]repositorio.PopInfo {
	rows, err := db.Query("SELECT id, ipv4, api_port, user, pass, status FROM sgp_pops")
	if err != nil {
		return nil
	}
	defer rows.Close()

	pops := make(map[int]repositorio.PopInfo)
	for rows.Next() {
		var p repositorio.PopInfo
		if err := rows.Scan(&p.ID, &p.IPv4, &p.APIPort, &p.User, &p.Pass, &p.Status); err != nil {
			continue
		}
		pops[p.ID] = p
	}
	return pops
}

// repararOfflineParaOnline corrige contratos marcados como Offline mas com
// sessao ativa na radacct.
func repararOfflineParaOnline(tag string, db *sql.DB) error {
	contratos, err := repositorio.BuscarContratosOfflineComSessao(db)
	if err != nil {
		return fmt.Errorf("erro ao buscar contratos offline: %w", err)
	}
	if contratos == nil {
		logger.Info(tag, "reparar_offline_para_online: nenhum contrato offline com sessao ativa")
		return nil
	}

	logger.Info(tag, "reparar_offline_para_online: %d contratos offline com sessao ativa", len(contratos))
	corrigidos := 0
	agora := fuso.Agora()
	for _, c := range contratos {
		if err := repositorio.AtualizarContratoOfflineParaOnline(db, c.ID,
			agora.Format("2006-01-02 15:04:05"),
			agora.Format("2006-01-02"),
			agora.Format("15:04:05")); err != nil {
			logger.Erro(tag, "erro reparar contrato %d: %v", c.ID, err)
			continue
		}
		logger.Info(tag, "contrato %d: Offline->Online (sessao ativa na radacct)", c.ID)
		corrigidos++
	}

	logger.Info(tag, "reparar_offline_para_online: %d contratos corrigidos para Online", corrigidos)
	return nil
}

// repararOnlineParaOffline corrige contratos marcados como Online mas sem
// atividade recente.
func repararOnlineParaOffline(tag string, db *sql.DB) error {
	limite := fuso.Agora().Add(-30 * time.Minute)
	contratos, err := repositorio.BuscarContratosResumo(db, limite.Format("2006-01-02 15:04:05"))
	if err != nil {
		return fmt.Errorf("erro ao buscar contratos online: %w", err)
	}

	corrigidos := 0
	for _, c := range contratos {
		logger.Info(tag, "contrato %d: Online->Offline (inativo >30min desde %s)", c.ID, limite.Format("2006-01-02 15:04:05"))

		if err := repositorio.AtualizarContratoForceOffline(db, c.ID); err != nil {
			logger.Erro(tag, "erro ao marcar offline contrato %d: %v", c.ID, err)
			continue
		}
		corrigidos++
	}

	logger.Info(tag, "reparar_online_para_offline: %d contratos corrigidos para Offline", corrigidos)
	return nil
}

// gerarToken gera um token hexadecimal aleatorio de 32 caracteres.
func gerarToken() string {
	return helpers.GerarToken()
}
