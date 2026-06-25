package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"gestor/internal/dominio"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
)

const tag = "gateway"

func HandleWebhook(w http.ResponseWriter, r *http.Request, instancia dominio.Instancia, rabbit *mensageria.RabbitMQ) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo nao permitido", http.StatusMethodNotAllowed)
		return
	}

	event := ""
	data := make(map[string]string)

	// Tenta parsear como form-urlencoded (PHP-style data[chave])
	if err := r.ParseForm(); err == nil {
		event = r.PostFormValue("event")
		for key, values := range r.Form {
			if strings.HasPrefix(key, "data[") && strings.HasSuffix(key, "]") {
				campo := key[5 : len(key)-1]
				if len(values) > 0 {
					data[campo] = values[0]
				}
			}
		}
	}

	// Se nao achou event ou data via form, tenta JSON
	if event == "" || len(data) == 0 {
		body, err := io.ReadAll(r.Body)
		if err == nil {
			var j struct {
				Event string            `json:"event"`
				Data  map[string]string `json:"data"`
			}
			if json.Unmarshal(body, &j) == nil && j.Event != "" {
				event = j.Event
				if len(j.Data) > 0 {
					data = j.Data
				}
				logger.Info(tag, "Instancia %d: webhook recebido via JSON", instancia.ID)
			}
		}
	}

	if event == "" {
		logger.Aviso(tag, "Instancia %d: event nao informado", instancia.ID)
		http.Error(w, "event nao informado", http.StatusBadRequest)
		return
	}
	if len(data) == 0 {
		logger.Aviso(tag, "Instancia %d: data nao informado", instancia.ID)
		http.Error(w, "data nao informado", http.StatusBadRequest)
		return
	}

	iuguID := data["id"]
	status := data["status"]
	payerName := data["payer_name"]
	logger.Info(tag, "Webhook: instancia=%d event=%s iugu_fatura=%s status=%s pagador=%s",
		instancia.ID, event, iuguID, status, payerName)

	if rabbit == nil {
		logger.Erro(tag, "Instancia %d: RabbitMQ nao disponivel", instancia.ID)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	msg := dominio.MensagemPagamentoIugu{
		Instancia: instancia,
		Event:     event,
		Data:      data,
		Tentativa: 0,
	}

	if err := rabbit.PublicarMensagem("processar_pagamento_iugu", msg); err != nil {
		logger.Erro(tag, "Instancia %d: erro ao publicar na fila: %v", instancia.ID, err)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	logger.Sucesso(tag, "Instancia %d: webhook publicado na fila processar_pagamento_iugu (fatura=%s)", instancia.ID, iuguID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("200"))
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
