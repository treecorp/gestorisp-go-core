package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gestor/internal/config"
	"gestor/internal/gateway"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
)

func main() {
	logger.Destaque("gateway", "Gateway de Pagamentos - Gestor ISP")
	logger.Info("gateway", "Inicializando...")

	cfg := config.Carregar()

	logger.Info("gateway", "Conectando ao banco global GISPADM...")
	if err := gateway.ConectarBancoGlobal(cfg.Banco); err != nil {
		logger.Erro("gateway", "Falha ao conectar no banco global: %v", err)
		os.Exit(1)
	}
	logger.Sucesso("gateway", "Conectado ao banco GISPADM")

	logger.Info("gateway", "Conectando ao RabbitMQ...")
	rabbit := mensageria.ConectarComRetry(cfg.RabbitMQ)
	defer rabbit.Fechar()
	logger.Sucesso("gateway", "Conectado ao RabbitMQ")

	servidor := gateway.NovoServidor(cfg, rabbit)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := servidor.Iniciar(); err != nil {
			logger.Erro("gateway", "Servidor HTTP encerrado: %v", err)
			quit <- syscall.SIGTERM
		}
	}()

	sinal := <-quit
	logger.Aviso("gateway", "Sinal recebido: %v. Encerrando...", sinal)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := servidor.Parar(ctx); err != nil {
		logger.Erro("gateway", "Erro ao encerrar servidor: %v", err)
	}

	logger.Info("gateway", "Gateway encerrado com sucesso")
}
