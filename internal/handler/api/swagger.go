package api

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/http"
	"text/template"

	"gestor/internal/infra/logger"
)

//go:embed openapi.yaml
var openapiYAML string

// handleOpenAPI serve o arquivo openapi.yaml com template aplicado.
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

// handleSwaggerUI serve a interface Swagger UI para explorar a API.
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
