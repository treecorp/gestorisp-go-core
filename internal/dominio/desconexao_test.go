package dominio

import (
	"testing"
	"time"
)

func TestMensagemDesconexaoExpirada_Nova(t *testing.T) {
	msg := MensagemDesconexaoContrato{
		CriadoEm: time.Now().Format(time.RFC3339),
	}
	if msg.Expirada() {
		t.Error("mensagem nova nao deveria estar expirada")
	}
}

func TestMensagemDesconexaoExpirada_24h(t *testing.T) {
	msg := MensagemDesconexaoContrato{
		CriadoEm: time.Now().Add(-25 * time.Hour).Format(time.RFC3339),
	}
	if !msg.Expirada() {
		t.Error("mensagem de 25h deveria estar expirada")
	}
}

func TestMensagemDesconexaoExpirada_CriadoEmVazio(t *testing.T) {
	msg := MensagemDesconexaoContrato{}
	if msg.Expirada() {
		t.Error("mensagem sem criado_em nao deveria estar expirada")
	}
}

func TestMensagemDesconexaoExpirada_FormatoInvalido(t *testing.T) {
	msg := MensagemDesconexaoContrato{
		CriadoEm: "formato-invalido",
	}
	if msg.Expirada() {
		t.Error("mensagem com formato invalido nao deveria estar expirada")
	}
}
