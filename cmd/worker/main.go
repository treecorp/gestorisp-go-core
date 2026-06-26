package main

import (
	"os"
	"os/signal"
	"syscall"

	"gestor/internal/config"
	"gestor/internal/entity"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
	"gestor/internal/handler/worker"
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

	go w.Iniciar([]worker.Consumidor{
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
			Fila: "listar_clientes_vencidos",
			Handler: func(instancia entity.Instancia) error {
				return worker.HandlerListarClientesVencidos(instancia, rabbit)
			},
		},
	})

	w.IniciarMensagem([]worker.ConsumidorMensagem{
		{
			Fila:          "processar_pagamento_iugu",
			Handler:       worker.HandlerProcessarPagamentoIugu,
			RetryInfinito: true,
		},
		{
			Fila:          "desconectar_contrato",
			Handler: func(body []byte, _ *mensageria.RabbitMQ) error {
				return worker.HandlerDesconectarContrato(body)
			},
			RetryInfinito: true,
		},
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sinal := <-quit
	logger.Aviso("worker", "Sinal recebido: %v. Encerrando...", sinal)
	logger.Info("worker", "Worker encerrado com sucesso")
}
