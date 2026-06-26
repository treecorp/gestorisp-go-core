package helpers

import "strings"

// LimparNumero remove todos os caracteres que não são dígitos de uma
// string de valor monetário, retornando apenas os números.
//
// Exemplo: "R$ 1.234,50" → "123450"
func LimparNumero(valor string) string {
	var sb strings.Builder
	sb.Grow(len(valor))
	for i := 0; i < len(valor); i++ {
		c := valor[i]
		if c >= '0' && c <= '9' {
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// FormatarMoeda converte um valor numérico em string para o formato
// monetário brasileiro com separador de milhar e vírgula decimal.
//
// Exemplo: "1234.50" → "1.234,50"
func FormatarMoeda(valor string) string {
	limpo := LimparNumero(valor)
	for len(limpo) < 3 {
		limpo = "0" + limpo
	}
	reais := limpo[:len(limpo)-2]
	centavos := limpo[len(limpo)-2:]

	// Adiciona separador de milhar a cada 3 dígitos
	var partes []string
	for len(reais) > 3 {
		partes = append([]string{reais[len(reais)-3:]}, partes...)
		reais = reais[:len(reais)-3]
	}
	if len(reais) > 0 {
		partes = append([]string{reais}, partes...)
	}
	reaisFormatado := strings.Join(partes, ".")

	return reaisFormatado + "," + centavos
}
