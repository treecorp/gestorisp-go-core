package helpers

import (
	"testing"
)

func TestFormatarMoeda(t *testing.T) {
	tests := []struct {
		name   string
		valor  string
		esperado string
	}{
		{"valor inteiro", "1234", "12,34"},
		{"valor com ponto decimal", "1234.50", "1.234,50"},
		{"valor com centavos unicos", "100.01", "100,01"},
		{"valor pequeno", "0.50", "0,50"},
		{"valor sem centavos", "1000", "10,00"},
		{"valor formatado BR", "R$ 1.234,56", "1.234,56"},
		{"string vazia", "", "0,00"},
		{"apenas centavos", ".99", "0,99"},
		{"milhao", "1000000.00", "1.000.000,00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatarMoeda(tt.valor)
			if got != tt.esperado {
				t.Errorf("FormatarMoeda(%q) = %q, esperado %q", tt.valor, got, tt.esperado)
			}
		})
	}
}

func TestLimparNumero(t *testing.T) {
	tests := []struct {
		name   string
		valor  string
		esperado string
	}{
		{"apenas numeros", "123450", "123450"},
		{"com ponto decimal", "1234.50", "123450"},
		{"formato BR completo", "R$ 1.234,56", "123456"},
		{"apenas simbolo", "R$ 100,00", "10000"},
		{"string vazia", "", ""},
		{"com espacos", "1 234,56", "123456"},
		{"apenas centavos", "0,99", "099"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LimparNumero(tt.valor)
			if got != tt.esperado {
				t.Errorf("LimparNumero(%q) = %q, esperado %q", tt.valor, got, tt.esperado)
			}
		})
	}
}
