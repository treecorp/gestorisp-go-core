package worker

import (
	"fmt"

	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
)

func HandlerRepairRadiusAcctstoptime(instancia dominio.Instancia) error {
	tag := "repair_radius_acctstoptime"
	logger.Inicio(tag, "Instancia %d: processando...", instancia.ID)

	db, err := banco.ConectarInstancia(
		instancia.EnvDBHost, instancia.EnvDBPort,
		instancia.EnvDBUser, instancia.EnvDBPass, instancia.EnvDBName,
	)
	if err != nil {
		return fmt.Errorf("falha ao conectar na instancia %d: %w", instancia.ID, err)
	}
	defer banco.FecharConexaoInstancia(db, tag)

	resultado, err := db.Exec("DELETE FROM radacct WHERE acctstoptime IS NULL")
	if err != nil {
		return fmt.Errorf("erro ao deletar registros orfaos: %w", err)
	}

	afetados, _ := resultado.RowsAffected()
	if afetados > 0 {
		logger.Sucesso(tag, "Instancia %d: %d registros removidos", instancia.ID, afetados)
	} else {
		logger.Info(tag, "Instancia %d: nenhum registro para limpar", instancia.ID)
	}

	return nil
}
