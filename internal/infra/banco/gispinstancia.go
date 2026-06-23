package banco

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"gestor/internal/config"
	"gestor/internal/dominio"
	"gestor/internal/infra/logger"
)

func ConectarInstancia(hostname, porta, username, password, database string) (*sql.DB, error) {
	if porta == "" {
		porta = "3306"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true",
		username, password, hostname, porta, database,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir conexao da instancia: %w", err)
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("falha ao pingar banco da instancia: %w", err)
	}

	return db, nil
}

func BuscarPopsOperacionais(db *sql.DB) ([]dominio.Pop, error) {
	query := `SELECT id, ipv4, api_port, user, pass, status, status_timeout, status_timeout_data_hora 
	           FROM sgp_pop WHERE status = 'OPERACIONAL' ORDER BY id ASC`

	linhas, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar POPs: %w", err)
	}
	defer linhas.Close()

	var pops []dominio.Pop
	for linhas.Next() {
		var p dominio.Pop
		if err := linhas.Scan(
			&p.ID, &p.IPv4, &p.APIPort, &p.User, &p.Pass,
			&p.Status, &p.StatusTimeout, &p.StatusTimeoutDataHora,
		); err != nil {
			return nil, fmt.Errorf("erro ao escanear POP: %w", err)
		}
		pops = append(pops, p)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("erro na iteracao dos POPs: %w", err)
	}

	return pops, nil
}

func AtualizarStatusTimeout(db *sql.DB, popID int, timeout int, dataHora *string) error {
	if dataHora != nil && *dataHora != "" {
		query := `UPDATE sgp_pop SET status_timeout = ?, status_timeout_data_hora = ? WHERE id = ?`
		_, err := db.Exec(query, timeout, *dataHora, popID)
		if err != nil {
			return fmt.Errorf("erro ao atualizar status_timeout do POP %d: %w", popID, err)
		}
	} else {
		query := `UPDATE sgp_pop SET status_timeout = ?, status_timeout_data_hora = NULL WHERE id = ?`
		_, err := db.Exec(query, timeout, popID)
		if err != nil {
			return fmt.Errorf("erro ao atualizar status_timeout do POP %d: %w", popID, err)
		}
	}

	return nil
}

func FecharConexaoInstancia(db *sql.DB, tag string) {
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Aviso(tag, "Erro ao fechar conexao da instancia: %v", err)
		}
	}
}

func PingInstancia(cfg config.BancoConfig) error {
	if pool == nil {
		return fmt.Errorf("pool de conexao nao inicializado")
	}
	if err := pool.Ping(); err != nil {
		return fmt.Errorf("pool gispadm sem conexao: %w", err)
	}
	return nil
}
