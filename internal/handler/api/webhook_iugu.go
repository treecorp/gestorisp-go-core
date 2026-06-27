package api

import (
	"net/http"
	"strings"

	"gestor/internal/handler/gateway"
	"gestor/internal/helpers"
	"gestor/internal/infra/logger"
)

// handlePagamentoIugu processa webhooks de pagamento Iugu recebidos
// via API unificada. Extrai o token da URL, autentica a instância e
// delega o processamento ao gateway antigo.
//

// gateway.HandleWebhook por uso direto de repositorio + service.
func (s *Servidor) handlePagamentoIugu(w http.ResponseWriter, r *http.Request) {
	logger.Info(tag, "Request recebido para %s", r.URL.Path)

	if r.Method != http.MethodPost {
		helpers.ResponderJSON(w, http.StatusMethodNotAllowed, helpers.RespostaJSON{Sucesso: false, Erro: "Metodo nao permitido"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/gateway/pagamentos/iugu/gatilho/")
	token := strings.TrimRight(path, "/")

	if token == "" {
		helpers.ResponderJSON(w, http.StatusBadRequest, helpers.RespostaJSON{Sucesso: false, Erro: "Token nao informado"})
		return
	}

	logger.Info(tag, "Autenticando token %s...", token)

	instancia, err := gateway.Autenticar(token, s.cfg.Banco, s.cfg)
	if err != nil {
		logger.Aviso(tag, "Token invalido: %s (%v)", token, err)
		helpers.ResponderJSON(w, http.StatusForbidden, helpers.RespostaJSON{Sucesso: false, Erro: "Nao permitido"})
		return
	}

	gateway.HandleWebhook(w, r, instancia, s.rabbit)
}
