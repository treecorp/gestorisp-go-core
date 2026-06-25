package worker

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"gestor/internal/dominio"
)

func TestHandlerDesconectarContrato(t *testing.T) {
	msg := dominio.MensagemDesconexaoContrato{
		Instancia: dominio.Instancia{
			ID:    1,
			Token: "teste-unitario",
		},
		ContratoID:  999,
		ClienteNome: "Teste Unitario",
		PPPoEUser:   "04720186475",
		PopIPv4:     "10.20.1.2",
		PopPort:     "8728",
		PopUser:     "api",
		PopPass:     "33223200#*",
		CriadoEm:    time.Now().Format(time.RFC3339),
	}

	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("erro ao serializar JSON: %v", err)
	}

	body := []byte(base64.StdEncoding.EncodeToString(jsonBytes))

	err = HandlerDesconectarContrato(body)
	if err != nil {
		t.Fatalf("Falha na desconexao: %v", err)
	}
}
