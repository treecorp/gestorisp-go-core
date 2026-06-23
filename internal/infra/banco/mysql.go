package banco

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"gestor/internal/config"
	"gestor/internal/infra/logger"
)

var pool *sql.DB

// Conectar inicializa o pool de conexoes MySQL
func Conectar(cfg config.BancoConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true",
		cfg.Usuario, cfg.Senha, cfg.Host, cfg.Porta, cfg.Database,
	)
	var err error
	pool, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir conexao: %w", err)
	}
	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(5 * time.Minute)

	if err := pool.Ping(); err != nil {
		return nil, fmt.Errorf("falha ao pingar banco: %w", err)
	}
	logger.Sucesso("banco", "Conectado ao MySQL GISPADM (%s:%s)", cfg.Host, cfg.Porta)
	return pool, nil
}

// ConectarComRetry tenta conectar ao MySQL em loop infinito com backoff progressivo.
// So retorna quando a conexao for estabelecida com sucesso.
// O backoff comeca em 2s e dobra ate o maximo de 60s entre tentativas.
func ConectarComRetry(cfg config.BancoConfig) *sql.DB {
	espera := 2 * time.Second
	for {
		db, err := Conectar(cfg)
		if err == nil {
			return db
		}
		logger.Aviso("banco", "Falha ao conectar: %v. Reintentando em %s...", err, espera)
		time.Sleep(espera)
		espera *= 2
		if espera > 60*time.Second {
			espera = 60 * time.Second
		}
	}
}

// Ping verifica se a conexao com o banco ainda esta ativa.
// Se falhar, tenta reconectar automaticamente.
func Ping(cfg config.BancoConfig) error {
	if pool == nil {
		return fmt.Errorf("pool de conexao nao inicializado")
	}
	if err := pool.Ping(); err != nil {
		logger.Aviso("banco", "Ping falhou: %v. Reconectando...", err)
		if _, err := Conectar(cfg); err != nil {
			return fmt.Errorf("falha ao reconectar: %w", err)
		}
		logger.Sucesso("banco", "Reconectado com sucesso")
	}
	return nil
}

// Fechar encerra o pool de conexoes
func Fechar() {
	if pool != nil {
		pool.Close()
		logger.Info("banco", "Conexao MySQL encerrada")
	}
}
