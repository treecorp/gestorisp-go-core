package worker

import (
	"fmt"

	"gestor/internal/entity"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
	"gestor/internal/repositorio"
)

// HandlerSyncConexoesRadiusArquivo migra registros da radacct para
// radacct_arquivo, processando registros com acctstoptime preenchido.
func HandlerSyncConexoesRadiusArquivo(instancia entity.Instancia) error {
	tag := "sync_conexoes_radius_arquivo"
	logger.Inicio(tag, "Instancia %d: processando...", instancia.ID)

	db, err := banco.ConectarInstancia(
		instancia.EnvDBHost, instancia.EnvDBPort,
		instancia.EnvDBUser, instancia.EnvDBPass, instancia.EnvDBName,
	)
	if err != nil {
		return fmt.Errorf("falha ao conectar na instancia %d: %w", instancia.ID, err)
	}
	defer banco.FecharConexaoInstancia(db, tag)

	colunasDisponiveis, err := repositorio.DetectarColunasArquivo(db)
	if err != nil {
		return fmt.Errorf("erro ao detectar colunas de radacct_arquivo: %w", err)
	}

	colunasRadacct, err := repositorio.DetectarColunasRadacct(db)
	if err != nil {
		return fmt.Errorf("erro ao detectar colunas de radacct: %w", err)
	}

	registros, err := repositorio.BuscarRadacctPendenteArquivo(db, colunasRadacct)
	if err != nil {
		return fmt.Errorf("erro ao buscar registros pendentes: %w", err)
	}

	if registros == nil {
		logger.Info(tag, "Instancia %d: nenhum registro pendente", instancia.ID)
		return nil
	}

	logger.Info(tag, "Instancia %d: %d registros pendentes", instancia.ID, len(registros))

	migrados := 0
	deletados := 0

	for _, rec := range registros {
		tx, err := db.Begin()
		if err != nil {
			logger.Erro(tag, "Instancia %d, radacctid %d: erro ao iniciar transacao: %v", instancia.ID, rec.RadAcctID, err)
			continue
		}

		if err := repositorio.ProcessarRegistro(tx, rec, colunasDisponiveis); err != nil {
			tx.Rollback()
			logger.Erro(tag, "Instancia %d, radacctid %d: %v", instancia.ID, rec.RadAcctID, err)
			continue
		}

		if err := tx.Commit(); err != nil {
			logger.Erro(tag, "Instancia %d, radacctid %d: erro ao commitar: %v", instancia.ID, rec.RadAcctID, err)
			continue
		}
		migrados++
		deletados++
	}

	logger.Sucesso(tag, "Instancia %d: %d migrados, %d deletados", instancia.ID, migrados, deletados)
	return nil
}
