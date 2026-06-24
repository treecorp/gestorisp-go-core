package worker

import "gestor/internal/infra/mensageria"

type ConsumidorMensagem struct {
	Fila    string
	Handler func([]byte, *mensageria.RabbitMQ) error
}
