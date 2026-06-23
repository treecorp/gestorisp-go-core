package banco

import (
	"database/sql"

	"gestor/internal/config"
	"gestor/internal/dominio"
	"gestor/internal/infra/logger"
)

// BuscarInstanciasAtivas retorna todas as instancias GISP-FULL com status Ativo
// Query original do PHP: SELECT id, token, env_dbname, env_dbuser, env_dbpass, env_dbhost
// FROM instancias WHERE app='GISP-FULL' AND status='Ativo'
func BuscarInstanciasAtivas(cfg config.BancoConfig) ([]dominio.Instancia, error) {
	if err := Ping(cfg); err != nil {
		return nil, err
	}

	query := `SELECT id, token, env_dbname, env_dbuser, env_dbpass, env_dbhost 
	           FROM instancias 
	           WHERE app = 'GISP-FULL' AND status = 'Ativo'`

	linhas, err := pool.Query(query)
	if err != nil {
		logger.Erro("gispadm", "Erro na consulta de instancias: %v", err)
		return nil, err
	}
	defer linhas.Close()

	var instancias []dominio.Instancia
	for linhas.Next() {
		var i dominio.Instancia
		if err := linhas.Scan(
			&i.ID, &i.Token, &i.EnvDBName, &i.EnvDBUser, &i.EnvDBPass, &i.EnvDBHost,
		); err != nil {
			return nil, err
		}
		instancias = append(instancias, i)
	}
	if err := linhas.Err(); err != nil {
		return nil, err
	}

	if len(instancias) == 0 {
		return nil, sql.ErrNoRows
	}
	return instancias, nil
}
