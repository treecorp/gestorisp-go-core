package helpers

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		max    int
		esperado string
	}{
		{"string menor que max", "abc", 5, "abc"},
		{"string igual ao max", "abcde", 5, "abcde"},
		{"string maior que max", "abcdef", 3, "abc..."},
		{"string vazia", "", 5, ""},
		{"max igual a zero", "abc", 0, "..."},
		{"unicode nao quebra", "café latte", 5, "café..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.max)
			if got != tt.esperado {
				t.Errorf("Truncate(%q, %d) = %q, esperado %q", tt.input, tt.max, got, tt.esperado)
			}
		})
	}
}
