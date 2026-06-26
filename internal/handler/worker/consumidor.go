package worker

import "gestor/internal/entity"

// Consumidor representa um consumidor de fila RabbitMQ baseado em instancia.
type Consumidor struct {
	Fila    string
	Handler func(entity.Instancia) error
}
