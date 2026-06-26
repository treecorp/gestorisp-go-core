package worker

import (
	"database/sql"
	"fmt"
	"time"

	"gestor/internal/entity"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/fuso"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
	"gestor/internal/repositorio"
	"gestor/internal/service/bloqueio"
)

// HandlerListarClientesVencidos processa a lista de clientes com faturas
// vencidas e publica mensagens de desconexao para os que devem ser bloqueados.
func HandlerListarClientesVencidos(instancia entity.Instancia, rabbit *mensageria.RabbitMQ) error {
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

	diasBloqueio := repositorio.LerDiasBloqueio(db)

	faturas, err := repositorio.BuscarFaturasVencidas(db, diasBloqueio)
	if err != nil {
		return fmt.Errorf("erro ao buscar faturas vencidas: %w", err)
	}
	if len(faturas) == 0 {
		logger.Info(tag, "Nenhuma fatura vencida encontrada")
		return nil
	}

	logger.Info(tag, "%d faturas vencidas para processar", len(faturas))

	// Wire repos for bloqueio service
	repos := &bloqueio.Repositorios{
		Contrato: &bloqueioContratoAdapter{},
		Bloqueio: &bloqueioBloqueioAdapter{},
	}

	var bloqueados []bloqueio.ClienteBloqueado
	for _, f := range faturas {
		bloqueado, err := bloqueio.ProcessarFatura(repos, tag, db, f, diasBloqueio)
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

	logger.Info(tag, "%d clientes para publicar na fila desconectar_contrato", len(bloqueados))

	pops := carregarPopsMap(db)
	for _, cb := range bloqueados {
		pop, ok := pops[cb.PopIPv4]
		if !ok {
			logger.Aviso(tag, "Cliente %s: POP nao encontrado", cb.PPPoEUser)
			continue
		}

		msg := entity.MensagemDesconexaoContrato{
			Instancia:   instancia,
			ContratoID:  cb.ContratoID,
			ClienteNome: cb.ClienteNome,
			PPPoEUser:   cb.PPPoEUser,
			PopIPv4:     pop.IPv4,
			PopPort:     pop.APIPort,
			PopUser:     pop.User,
			PopPass:     pop.Pass,
			CriadoEm:    fuso.Agora().Format(time.RFC3339),
		}

		if err := rabbit.PublicarMensagem("desconectar_contrato", msg); err != nil {
			logger.Erro(tag, "Erro ao publicar desconexao para %s: %v", cb.PPPoEUser, err)
			continue
		}
		logger.Sucesso(tag, "Publicada desconexao para %s (contrato %d)", cb.PPPoEUser, cb.ContratoID)
	}

	logger.Sucesso(tag, "Bloqueio realizado com sucesso para %d contratos", len(bloqueados))
	return nil
}

// carregarPopsMap carrega os POPs operacionais e retorna um mapa indexado por IPv4.
func carregarPopsMap(db *sql.DB) map[string]entity.Pop {
	pops, err := repositorio.BuscarPopsOperacionais(db)
	if err != nil {
		return nil
	}
	m := make(map[string]entity.Pop)
	for _, p := range pops {
		m[p.IPv4] = p
	}
	return m
}
