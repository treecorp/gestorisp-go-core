package gateway

import (
	"database/sql"
	"fmt"

	"gestor/internal/config"
	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
)

func Autenticar(token string, cfg config.BancoConfig, configGeral *config.Config) (dominio.Instancia, error) {
	tag := "gateway"
	instancia, err := banco.BuscarInstanciaPorToken(token, cfg, configGeral)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Aviso(tag, "Token nao encontrado: %s", token)
			return dominio.Instancia{}, fmt.Errorf("token invalido: %s", token)
		}
		logger.Erro(tag, "Erro ao buscar instancia por token %s: %v", token, err)
		return dominio.Instancia{}, fmt.Errorf("erro ao autenticar token: %w", err)
	}
	logger.Sucesso(tag, "Instancia %d autenticada: %s", instancia.ID, instancia.EnvDBName)
	return instancia, nil
}
