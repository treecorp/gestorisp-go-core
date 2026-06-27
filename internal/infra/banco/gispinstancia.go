package banco

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

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

func FecharConexaoInstancia(db *sql.DB, tag string) {
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Aviso(tag, "Erro ao fechar conexao da instancia: %v", err)
		}
	}
}


