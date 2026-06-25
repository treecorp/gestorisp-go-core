package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"gestor/internal/config"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
)

type Servidor struct {
	cfg     *config.Config
	servico *http.Server
	rabbit  *mensageria.RabbitMQ
}

func NovoServidor(cfg *config.Config, rabbit *mensageria.RabbitMQ) *Servidor {
	return &Servidor{
		cfg:    cfg,
		rabbit: rabbit,
	}
}

func (s *Servidor) Iniciar() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/routeros/desconectarpppoe", s.handleDesconectarPPPoE)

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

func (s *Servidor) Parar(ctx context.Context) error {
	logger.Aviso(tag, "Encerrando servidor HTTP...")
	return s.servico.Shutdown(ctx)
}

func (s *Servidor) handleDesconectarPPPoE(w http.ResponseWriter, r *http.Request) {
	HandleDesconectarPPPoE(w, r, s.rabbit)
}
