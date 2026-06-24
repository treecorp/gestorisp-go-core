package dominio

type MensagemPagamentoIugu struct {
	Instancia Instancia          `json:"instancia"`
	Event     string             `json:"event"`
	Data      map[string]string  `json:"data"`
	Tentativa int                `json:"tentativa"`
}
