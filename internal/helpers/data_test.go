package helpers

import (
	"testing"
)

func TestExtrairData(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		esperado string
	}{
		{"timestamp completo", "2024-01-15 10:30:00", "2024-01-15"},
		{"apenas data", "2024-01-15", "2024-01-15"},
		{"string vazia", "", ""},
		{"formato diferente", "2024/01/15 10:30:00", "2024/01/15"},
		{"apenas ano", "2024", "2024"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtrairData(tt.input)
			if got != tt.esperado {
				t.Errorf("ExtrairData(%q) = %q, esperado %q", tt.input, got, tt.esperado)
			}
		})
	}
}

func TestExtrairHora(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		esperado string
	}{
		{"timestamp completo", "2024-01-15 10:30:00", "10:30:00"},
		{"apenas data", "2024-01-15", ""},
		{"string vazia", "", ""},
		{"meia-noite", "2024-01-15 00:00:00", "00:00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtrairHora(tt.input)
			if got != tt.esperado {
				t.Errorf("ExtrairHora(%q) = %q, esperado %q", tt.input, got, tt.esperado)
			}
		})
	}
}
