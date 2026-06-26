package helpers

import (
	"testing"
)

func TestGerarProtocolo(t *testing.T) {
	// Reseta o contador para o teste
	idCounter = 0

	// Testa range 100000-999999 (usado para protocolo de baixa)
	for i := 0; i < 100; i++ {
		proto := GerarProtocolo(100000, 999999)
		if proto < 100000 || proto > 999999 {
			t.Errorf("GerarProtocolo(100000, 999999) = %d, fora do intervalo [100000, 999999]", proto)
		}
	}

	// Testa range 300000-399999 (usado para registro de baixa)
	idCounter = 0
	for i := 0; i < 100; i++ {
		proto := GerarProtocolo(300000, 399999)
		if proto < 300000 || proto > 399999 {
			t.Errorf("GerarProtocolo(300000, 399999) = %d, fora do intervalo [300000, 399999]", proto)
		}
	}

	// Testa range 400000-499999 (usado para desbloqueio)
	idCounter = 0
	for i := 0; i < 100; i++ {
		proto := GerarProtocolo(400000, 499999)
		if proto < 400000 || proto > 499999 {
			t.Errorf("GerarProtocolo(400000, 499999) = %d, fora do intervalo [400000, 499999]", proto)
		}
	}

	// Testa valores iguais (min == max)
	proto := GerarProtocolo(42, 42)
	if proto != 42 {
		t.Errorf("GerarProtocolo(42, 42) = %d, esperado 42", proto)
	}
}
