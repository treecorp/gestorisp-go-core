package api

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"gestor/internal/config"
	"gestor/internal/gateway"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
)

//go:embed openapi.yaml
var openapiYAML string

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

func (s *Servidor) Parar(ctx context.Context) error {
	logger.Aviso(tag, "Encerrando servidor HTTP...")
	return s.servico.Shutdown(ctx)
}

func (s *Servidor) handleDesconectarPPPoE(w http.ResponseWriter, r *http.Request) {
	HandleDesconectarPPPoE(w, r, s.rabbit)
}

func (s *Servidor) handlePagamentoIugu(w http.ResponseWriter, r *http.Request) {
	logger.Info(tag, "Request recebido para %s", r.URL.Path)

	if r.Method != http.MethodPost {
		responderJSON(w, http.StatusMethodNotAllowed, resposta{Sucesso: false, Erro: "Metodo nao permitido"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/gateway/pagamentos/iugu/gatilho/")
	token := strings.TrimRight(path, "/")

	if token == "" {
		responderJSON(w, http.StatusBadRequest, resposta{Sucesso: false, Erro: "Token nao informado"})
		return
	}

	logger.Info(tag, "Autenticando token %s...", token)

	instancia, err := gateway.Autenticar(token, s.cfg.Banco, s.cfg)
	if err != nil {
		logger.Aviso(tag, "Token invalido: %s (%v)", token, err)
		responderJSON(w, http.StatusForbidden, resposta{Sucesso: false, Erro: "Nao permitido"})
		return
	}

	gateway.HandleWebhook(w, r, instancia, s.rabbit)
}

func (s *Servidor) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	tpl, err := template.New("openapi").Parse(openapiYAML)
	if err != nil {
		logger.Erro(tag, "Erro ao fazer parse do template openapi.yaml: %v", err)
		responderJSON(w, http.StatusInternalServerError, resposta{Sucesso: false, Erro: "Erro interno"})
		return
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, map[string]string{
		"ServerURL": fmt.Sprintf("%s://%s", scheme, host),
	}); err != nil {
		logger.Erro(tag, "Erro ao renderizar template openapi.yaml: %v", err)
		responderJSON(w, http.StatusInternalServerError, resposta{Sucesso: false, Erro: "Erro interno"})
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

func (s *Servidor) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <title>API Gestor ISP - Swagger</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({ url: "/openapi.yaml", dom_id: "#swagger-ui" })
  </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func (s *Servidor) handleNotFound(w http.ResponseWriter, r *http.Request) {
	responderJSON(w, http.StatusNotFound, resposta{Sucesso: false, Erro: "Rota nao encontrada"})
}
