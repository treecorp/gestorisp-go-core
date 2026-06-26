package helpers

// Truncate trunca uma string para o tamanho máximo informado, adicionando
// "..." ao final caso a string original exceda o limite.
//
// Se a string for menor ou igual ao limite, retorna a string original.
func Truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
