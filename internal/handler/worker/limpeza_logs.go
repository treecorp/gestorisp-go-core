package worker

import (
	"fmt"

	"gestor/internal/entity"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
)

var tabelasLimpeza = []string{
	"sgp_clientes_logs",
	"sgp_monitor_interfaces_historico",
	"sgp_webservices_cron",
	"radpostauth",
	"SystemEvents",
}

// HandlerLimpezaLogs executa o truncate de tabelas de log para uma instancia.
func HandlerLimpezaLogs(instancia entity.Instancia) error {
	tag := "limpeza_logs"
	logger.Inicio(tag, "Instancia %d: processando...", instancia.ID)

	db, err := banco.ConectarInstancia(
		instancia.EnvDBHost, instancia.EnvDBPort,
		instancia.EnvDBUser, instancia.EnvDBPass, instancia.EnvDBName,
	)
	if err != nil {
		return fmt.Errorf("falha ao conectar na instancia %d: %w", instancia.ID, err)
	}
	defer banco.FecharConexaoInstancia(db, tag)

	for _, tabela := range tabelasLimpeza {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", tabela))
		if err != nil {
			logger.Erro(tag, "Instancia %d: erro ao truncar %s: %v", instancia.ID, tabela, err)
			continue
		}
		logger.Sucesso(tag, "Instancia %d: %s truncada", instancia.ID, tabela)
	}

	return nil
}
