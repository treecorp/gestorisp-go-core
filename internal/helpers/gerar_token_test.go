package helpers

import (
	"testing"
)

func TestGerarToken(t *testing.T) {
	// Gera varios tokens e verifica propriedades basicas
	for i := 0; i < 10; i++ {
		token := GerarToken()
		if token == "" {
			t.Fatal("GerarToken() retornou string vazia")
		}
		if len(token) != 32 {
			t.Errorf("GerarToken() len = %d, esperado 32", len(token))
		}
		// Verifica se contem apenas caracteres hexadecimais
		for _, c := range token {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("GerarToken() caractere invalido: %c", c)
			}
		}
	}

	// Gera dois tokens e verifica se sao diferentes
	t1 := GerarToken()
	t2 := GerarToken()
	if t1 == t2 {
		t.Error("GerarToken() gerou tokens iguais consecutivamente")
	}
}
