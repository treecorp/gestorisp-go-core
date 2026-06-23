package main

import (
	"os"
	"os/signal"
	"syscall"

	"gestor/internal/config"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
	"gestor/internal/worker"
)

func main() {
	logger.Destaque("worker", "Worker de processamento - Gestor ISP")
	logger.Info("worker", "Inicializando...")

	cfg := config.Carregar()

	logger.Info("worker", "Aguardando conexao com RabbitMQ...")
	rabbit := mensageria.ConectarComRetry(cfg.RabbitMQ)
	defer rabbit.Fechar()

	logger.Sucesso("worker", "Conectado ao RabbitMQ. Iniciando consumidores...")

	w := worker.NovoWorker(cfg, rabbit)
	w.Iniciar([]worker.Consumidor{
		{
			Fila:    "check_pop_status",
			Handler: worker.HandlerCheckPopStatus,
		},
		{
			Fila:    "run_cluster",
			Handler: worker.HandlerRunCluster,
		},
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sinal := <-quit
	logger.Aviso("worker", "Sinal recebido: %v. Encerrando...", sinal)
	logger.Info("worker", "Worker encerrado com sucesso")
}
