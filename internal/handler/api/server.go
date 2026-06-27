package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"gestor/internal/config"
	"gestor/internal/helpers"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
)

const tag = "api"

// Servidor gerencia o servidor HTTP da API REST unificada.
type Servidor struct {
	cfg     *config.Config
	servico *http.Server
	rabbit  *mensageria.RabbitMQ
}

// NovoServidor cria uma nova instância do servidor da API.
func NovoServidor(cfg *config.Config, rabbit *mensageria.RabbitMQ) *Servidor {
	return &Servidor{
		cfg:    cfg,
		rabbit: rabbit,
	}
}

// Iniciar inicia o servidor HTTP na porta configurada e registra as rotas.
func (s *Servidor) Iniciar() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/routeros/desconectarpppoe", s.handleDesconectarPPPoE)
	mux.HandleFunc("/api/v2/gateway/pagamentos/iugu/gatilho/", s.handlePagamentoIugu)
	mux.HandleFunc("/openapi.yaml", s.handleOpenAPI)
	mux.HandleFunc("/swagger", s.handleSwaggerUI)
	mux.HandleFunc("/", s.handleNotFound)

	addr := fmt.Sprintf(":%s", s.cfg.APIPort)
	s.servico = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	logger.Sucesso(tag, "Servidor HTTP ouvindo na porta %s", s.cfg.APIPort)
	return s.servico.ListenAndServe()
}

// Parar encerra graciosamente o servidor HTTP.
func (s *Servidor) Parar(ctx context.Context) error {
	logger.Aviso(tag, "Encerrando servidor HTTP...")
	return s.servico.Shutdown(ctx)
}

// handleDesconectarPPPoE delega o processamento de desconexão PPPoE
// para a função HandleDesconectarPPPoE no mesmo pacote.
func (s *Servidor) handleDesconectarPPPoE(w http.ResponseWriter, r *http.Request) {
	HandleDesconectarPPPoE(w, r, s.rabbit)
}

// handleNotFound retorna 404 para rotas não encontradas.
func (s *Servidor) handleNotFound(w http.ResponseWriter, r *http.Request) {
	helpers.ResponderJSON(w, http.StatusNotFound, helpers.RespostaJSON{Sucesso: false, Erro: "Rota nao encontrada"})
}
