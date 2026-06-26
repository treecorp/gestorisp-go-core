package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gestor/internal/handler/api"
	"gestor/internal/config"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
)

func main() {
	logger.Destaque("api", "API Gestor ISP - RouterOS PPPoE + Gateway Iugu")
	logger.Info("api", "Inicializando...")

	cfg := config.Carregar()

	logger.Info("api", "Conectando ao banco global GISPADM...")
	if _, err := banco.Conectar(cfg.Banco); err != nil {
		logger.Erro("api", "Falha ao conectar no banco global: %v", err)
		os.Exit(1)
	}
	logger.Sucesso("api", "Conectado ao banco GISPADM")

	logger.Info("api", "Conectando ao RabbitMQ...")
	rabbit := mensageria.ConectarComRetry(cfg.RabbitMQ)
	defer rabbit.Fechar()
	logger.Sucesso("api", "Conectado ao RabbitMQ")

	servidor := api.NovoServidor(cfg, rabbit)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := servidor.Iniciar(); err != nil {
			logger.Erro("api", "Servidor HTTP encerrado: %v", err)
			quit <- syscall.SIGTERM
		}
	}()

	sinal := <-quit
	logger.Aviso("api", "Sinal recebido: %v. Encerrando...", sinal)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := servidor.Parar(ctx); err != nil {
		logger.Erro("api", "Erro ao encerrar servidor: %v", err)
	}

	logger.Info("api", "API encerrada com sucesso")
}
