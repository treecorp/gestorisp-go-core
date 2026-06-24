package main

import (
	"os"
	"os/signal"
	"syscall"

	"gestor/internal/config"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
	"gestor/internal/infra/observabilidade"
	"gestor/internal/worker"
)

func main() {
	logger.Destaque("worker", "Worker de processamento - Gestor ISP")
	logger.Info("worker", "Inicializando...")

	cfg := config.Carregar()

	if cfg.DashboardIngestURL != "" {
		observabilidade.ConfigurarIngestor(cfg.DashboardIngestURL)
		observabilidade.DefinirServico("worker")
		ingestor := observabilidade.LogIngestor{}
		logger.AdicionarHook(ingestor.WriteLog)
		logger.Info("worker", "Ingestor de logs configurado: %s", cfg.DashboardIngestURL)
	}

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
		{
			Fila:    "sync_conexoes_radius_arquivo",
			Handler: worker.HandlerSyncConexoesRadiusArquivo,
		},
		{
			Fila:    "cron_1",
			Handler: worker.HandlerCron1,
		},
		{
			Fila:    "repair_radius_acctstoptime",
			Handler: worker.HandlerRepairRadiusAcctstoptime,
		},
		{
			Fila:    "limpeza_logs",
			Handler: worker.HandlerLimpezaLogs,
		},
		{
			Fila:    "listar_clientes_vencidos",
			Handler: worker.HandlerListarClientesVencidos,
		},
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sinal := <-quit
	logger.Aviso("worker", "Sinal recebido: %v. Encerrando...", sinal)
	logger.Info("worker", "Worker encerrado com sucesso")
}
