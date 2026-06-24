package main

import (
	"net/http"

	"gestor/internal/config"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/observabilidade"
)

func main() {
	logger.Destaque("dashboard", "Dashboard de Logs ao Vivo - Gestor ISP")
	logger.Info("dashboard", "Inicializando...")

	cfg := config.Carregar()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/ingest", observabilidade.HandlerIngest)
	mux.HandleFunc("/api/events", observabilidade.HandlerSSE)
	mux.HandleFunc("/api/metricas", observabilidade.HandlerMetricas)
	mux.Handle("/", http.FileServer(http.Dir("web/dashboard")))

	addr := ":" + cfg.DashboardPort
	logger.Sucesso("dashboard", "Servidor iniciado em %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Erro("dashboard", "Falha ao iniciar servidor: %v", err)
	}
}
