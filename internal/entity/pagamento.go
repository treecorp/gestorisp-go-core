package entity

// MensagemPagamentoIugu representa a mensagem publicada na fila RabbitMQ
// quando um webhook de pagamento Iugu é recebido.
type MensagemPagamentoIugu struct {
	Instancia Instancia          `json:"instancia"`
	Event     string             `json:"event"`
	Data      map[string]string  `json:"data"`
	Tentativa int                `json:"tentativa"`
}
