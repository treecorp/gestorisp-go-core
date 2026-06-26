package helpers

// idCounter é um contador global usado para gerar números de protocolo
// únicos dentro do mesmo processo.
var idCounter int

// GerarProtocolo gera um número de protocolo aleatório dentro do intervalo
// [min, max] usando um contador sequencial combinado com o range.
func GerarProtocolo(min, max int) int {
	idCounter++
	return min + (idCounter % (max - min + 1))
}
