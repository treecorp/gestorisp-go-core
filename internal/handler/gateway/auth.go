package gateway

import (
	"database/sql"
	"fmt"

	"gestor/internal/config"
	"gestor/internal/entity"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
)

// Autenticar busca a instancia GISP associada ao token informado.
// Retorna a instancia se o token for valido, ou erro caso contrario.
func Autenticar(token string, cfg config.BancoConfig, configGeral *config.Config) (entity.Instancia, error) {
	tag := "gateway"
	instancia, err := banco.BuscarInstanciaPorToken(token, cfg, configGeral)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Aviso(tag, "Token nao encontrado: %s", token)
			return entity.Instancia{}, fmt.Errorf("token invalido: %s", token)
		}
		logger.Erro(tag, "Erro ao buscar instancia por token %s: %v", token, err)
		return entity.Instancia{}, fmt.Errorf("erro ao autenticar token: %w", err)
	}
	logger.Sucesso(tag, "Instancia %d autenticada: %s", instancia.ID, instancia.EnvDBName)
	return instancia, nil
}

