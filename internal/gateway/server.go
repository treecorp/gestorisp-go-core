package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gestor/internal/config"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
)

type Servidor struct {
	cfg     *config.Config
	servico *http.Server
}

func NovoServidor(cfg *config.Config) *Servidor {
	return &Servidor{
		cfg: cfg,
	}
}

func (s *Servidor) Iniciar() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/pagamentos/iugu/gatilho/", s.handleGatilho)

	addr := fmt.Sprintf(":%s", s.cfg.GatewayPort)
	s.servico = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	logger.Sucesso("gateway", "Servidor HTTP ouvindo na porta %s", s.cfg.GatewayPort)
	return s.servico.ListenAndServe()
}

func (s *Servidor) Parar(ctx context.Context) error {
	logger.Aviso("gateway", "Encerrando servidor HTTP...")
	return s.servico.Shutdown(ctx)
}

func (s *Servidor) handleGatilho(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/pagamentos/iugu/gatilho/")
	token := strings.TrimRight(path, "/")

	if token == "" {
		http.Error(w, "Token nao informado", http.StatusBadRequest)
		return
	}

	logger.Info("gateway", "Request recebido para token %s", token)

	instancia, err := Autenticar(token, s.cfg.Banco, s.cfg)
	if err != nil {
		logger.Aviso("gateway", "Token invalido: %s", token)
		http.Error(w, "Nao permitido", http.StatusForbidden)
		return
	}

	HandleWebhook(w, r, instancia)
}

func PingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("200"))
}

func ConectarBancoGlobal(cfg config.BancoConfig) error {
	_, err := banco.Conectar(cfg)
	return err
}
