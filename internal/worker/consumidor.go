package worker

import "gestor/internal/dominio"

type Consumidor struct {
	Fila    string
	Handler func(dominio.Instancia) error
}
