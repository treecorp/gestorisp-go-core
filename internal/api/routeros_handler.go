package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gestor/internal/dominio"
	"gestor/internal/infra/fuso"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
)

const tag = "api"

type SolicitacaoDesconectarPPPoE struct {
	InstanciaID int    `json:"instancia_id"`
	ContratoID  int    `json:"contrato_id"`
	ClienteNome string `json:"cliente_nome"`
	PPPoEUser   string `json:"pppoe_user"`
	PopIPv4     string `json:"pop_ipv4"`
	PopPort     string `json:"pop_port"`
	PopUser     string `json:"pop_user"`
	PopPass     string `json:"pop_pass"`
}

type resposta struct {
	Sucesso  bool   `json:"sucesso"`
	Mensagem string `json:"mensagem,omitempty"`
	Erro     string `json:"erro,omitempty"`
}

func HandleDesconectarPPPoE(w http.ResponseWriter, r *http.Request, rabbit *mensageria.RabbitMQ) {
	if r.Method != http.MethodPost {
		responderJSON(w, http.StatusMethodNotAllowed, resposta{Sucesso: false, Erro: "Metodo nao permitido"})
		return
	}

	var req SolicitacaoDesconectarPPPoE
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Aviso(tag, "JSON invalido: %v", err)
		responderJSON(w, http.StatusBadRequest, resposta{Sucesso: false, Erro: fmt.Sprintf("JSON invalido: %v", err)})
		return
	}

	if req.PPPoEUser == "" {
		responderJSON(w, http.StatusBadRequest, resposta{Sucesso: false, Erro: "pppoe_user é obrigatorio"})
		return
	}
	if req.PopIPv4 == "" {
		responderJSON(w, http.StatusBadRequest, resposta{Sucesso: false, Erro: "pop_ipv4 é obrigatorio"})
		return
	}
	if req.PopPort == "" {
		responderJSON(w, http.StatusBadRequest, resposta{Sucesso: false, Erro: "pop_port é obrigatorio"})
		return
	}
	if req.PopUser == "" {
		responderJSON(w, http.StatusBadRequest, resposta{Sucesso: false, Erro: "pop_user é obrigatorio"})
		return
	}
	if req.PopPass == "" {
		responderJSON(w, http.StatusBadRequest, resposta{Sucesso: false, Erro: "pop_pass é obrigatorio"})
		return
	}

	msg := dominio.MensagemDesconexaoContrato{
		Instancia:   dominio.Instancia{ID: req.InstanciaID},
		ContratoID:  req.ContratoID,
		ClienteNome: req.ClienteNome,
		PPPoEUser:   req.PPPoEUser,
		PopIPv4:     req.PopIPv4,
		PopPort:     req.PopPort,
		PopUser:     req.PopUser,
		PopPass:     req.PopPass,
		CriadoEm:    fuso.Agora().Format(time.RFC3339),
	}

	if err := rabbit.PublicarMensagem("desconectar_contrato", msg); err != nil {
		logger.Erro(tag, "Erro ao publicar na fila desconectar_contrato: %v", err)
		responderJSON(w, http.StatusInternalServerError, resposta{Sucesso: false, Erro: fmt.Sprintf("Erro ao publicar na fila: %v", err)})
		return
	}

	logger.Sucesso(tag, "Instancia %d: desconexao do contrato %d (%s) publicada na fila (pppoe=%s, pop=%s:%s)",
		req.InstanciaID, req.ContratoID, req.ClienteNome, req.PPPoEUser, req.PopIPv4, req.PopPort)

	responderJSON(w, http.StatusOK, resposta{Sucesso: true, Mensagem: "Publicado na fila desconectar_contrato"})
}

func responderJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
