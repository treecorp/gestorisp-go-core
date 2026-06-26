package helpers

// ExtrairData extrai a parte da data (YYYY-MM-DD) de um timestamp completo
// no formato "YYYY-MM-DD HH:MM:SS".
//
// Exemplo: "2024-01-15 10:30:00" → "2024-01-15"
// Se a string for vazia ou menor que 10 caracteres, retorna a string original.
func ExtrairData(dt string) string {
	if len(dt) >= 10 {
		return dt[:10]
	}
	return dt
}

// ExtrairHora extrai a parte da hora (HH:MM:SS) de um timestamp completo
// no formato "YYYY-MM-DD HH:MM:SS".
//
// Exemplo: "2024-01-15 10:30:00" → "10:30:00"
// Se a string for menor que 19 caracteres, retorna string vazia.
func ExtrairHora(dt string) string {
	if len(dt) >= 19 {
		return dt[11:19]
	}
	return ""
}
