package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gestor/internal/config"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
)

// Servidor gerencia o servidor HTTP que recebe webhooks de pagamento Iugu
// diretamente na porta de gateway (8082).
type Servidor struct {
	cfg     *config.Config
	servico *http.Server
	rabbit  *mensageria.RabbitMQ
}

// NovoServidor cria uma nova instância do servidor de gateway.
func NovoServidor(cfg *config.Config, rabbit *mensageria.RabbitMQ) *Servidor {
	return &Servidor{
		cfg:    cfg,
		rabbit: rabbit,
	}
}

// Iniciar inicia o servidor HTTP na porta configurada.
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

// Parar encerra graciosamente o servidor HTTP.
func (s *Servidor) Parar(ctx context.Context) error {
	logger.Aviso("gateway", "Encerrando servidor HTTP...")
	return s.servico.Shutdown(ctx)
}

// handleGatilho processa requisições de webhook Iugu.
func (s *Servidor) handleGatilho(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/pagamentos/iugu/gatilho/")
	token := strings.TrimRight(path, "/")

	if token == "" {
		responderJSON(w, http.StatusBadRequest, respostaJSON{Sucesso: false, Erro: "Token nao informado"})
		return
	}

	logger.Info("gateway", "Request recebido para token %s", token)

	instancia, err := Autenticar(token, s.cfg.Banco, s.cfg)
	if err != nil {
		logger.Aviso("gateway", "Token invalido: %s", token)
		responderJSON(w, http.StatusForbidden, respostaJSON{Sucesso: false, Erro: "Nao permitido"})
		return
	}

	HandleWebhook(w, r, instancia, s.rabbit)
}

// PingHandler é um health check simples.
func PingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("200"))
}

type respostaJSON struct {
	Sucesso  bool   `json:"sucesso"`
	Mensagem string `json:"mensagem,omitempty"`
	Erro     string `json:"erro,omitempty"`
}

func responderJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
