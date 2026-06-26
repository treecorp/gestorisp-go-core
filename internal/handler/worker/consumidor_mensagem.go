package worker

import "gestor/internal/infra/mensageria"

// ConsumidorMensagem representa um consumidor de fila RabbitMQ baseado em mensagem binaria.
type ConsumidorMensagem struct {
	Fila          string
	Handler       func([]byte, *mensageria.RabbitMQ) error
	RetryInfinito bool
}
