package main

import (
	"os"
	"os/signal"
	"syscall"

	"gestor/internal/config"
	"gestor/internal/cron"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
	"gestor/internal/infra/observabilidade"
)

func main() {
	logger.Destaque("gestor", "Backend Unificado - Gestor ISP")
	logger.Info("gestor", "Inicializando...")

	// --- 1. Carregar configuracoes ---
	cfg := config.Carregar()

	// --- 2. Configurar ingestao de logs para o dashboard ---
	if cfg.DashboardIngestURL != "" {
		observabilidade.ConfigurarIngestor(cfg.DashboardIngestURL)
		observabilidade.DefinirServico("gestor")
		ingestor := observabilidade.LogIngestor{}
		logger.AdicionarHook(ingestor.WriteLog)
		logger.Info("gestor", "Ingestor de logs configurado: %s", cfg.DashboardIngestURL)
	}

	// --- 3. Conectar no banco GISPADM (loop infinito ate conseguir) ---
	logger.Info("gestor", "Aguardando conexao com banco GISPADM...")
	banco.ConectarComRetry(cfg.Banco)
	defer banco.Fechar()

	// --- 4. Conectar no RabbitMQ (loop infinito ate conseguir) ---
	logger.Info("gestor", "Aguardando conexao com RabbitMQ...")
	rabbit := mensageria.ConectarComRetry(cfg.RabbitMQ)
	defer rabbit.Fechar()

	logger.Sucesso("gestor", "Conexoes estabelecidas. Iniciando agendador...")

	// --- 5. Configurar agendador com as tarefas cron ---
	agendador := cron.NovoAgendador(cfg, rabbit)
	agendador.Iniciar([]cron.TarefaRegistro{
		{Expressao: "0 */1 * * * *", Nome: "cron_um", Fila: "cron_1"},
		{Expressao: "0 */1 * * * *", Nome: "executar_cluster", Fila: "run_cluster"},
		{Expressao: "*/30 * * * * *", Nome: "verificar_status_pop", Fila: "check_pop_status"},
		{Expressao: "0 */1 * * * *", Nome: "sincronizar_conexoes", Fila: "sync_conexoes_radius_arquivo"},
		{Expressao: "0 30 0 * * *", Nome: "reparar_radius", Fila: "repair_radius_acctstoptime"},
		{Expressao: "0 30 0 * * *", Nome: "limpeza_logs", Fila: "limpeza_logs"},
		{Expressao: "0 10 14 * * *", Nome: "listar_clientes_vencidos", Fila: "listar_clientes_vencidos"},
	})

	// --- 6. Aguardar sinal de encerramento ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sinal := <-quit
	logger.Aviso("gestor", "Sinal recebido: %v. Encerrando...", sinal)
	agendador.Parar()
	logger.Info("gestor", "Gestor encerrado com sucesso")
}
