package entity

import (
	"database/sql"
	"time"
)

// Fatura representa uma fatura de cliente no sistema GISP.
//
// Mapeia os campos da tabela sgp_clientes_faturas com métodos
// para verificação de status e cálculo de atraso.
type Fatura struct {
	ID             int            `json:"id"`
	Token          string         `json:"token"`
	FaturaID       string         `json:"fatura_id"`
	Valor          string         `json:"valor"`
	ValorPago      string         `json:"valor_pago"`
	DataVencimento string         `json:"data_vencimento"`
	DataPagamento  string         `json:"data_pagamento"`
	Status         string         `json:"status"`
	ContratoID     int            `json:"contrato_id"`
	ClienteToken   string         `json:"cliente_token"`
	ContratoToken  string         `json:"contrato_token"`
	GatewayID      sql.NullInt64  `json:"gateway_id"`
}

// EstaPaga retorna true se o status da fatura indicar pagamento.
func (f Fatura) EstaPaga() bool {
	return f.Status == "Pago"
}

// CalcularDiasAtraso calcula a quantidade de dias entre a data de
// vencimento e a data atual. Retorna 0 se a fatura estiver paga ou
// se a data de vencimento não puder ser interpretada.
func (f Fatura) CalcularDiasAtraso() int {
	if f.EstaPaga() {
		return 0
	}
	if f.DataVencimento == "" {
		return 0
	}

	// Aceita formatos YYYY-MM-DD ou YYYY-MM-DD HH:MM:SS
	vencimento, err := time.Parse("2006-01-02", f.DataVencimento[:10])
	if err != nil {
		return 0
	}

	dias := int(time.Since(vencimento).Hours() / 24)
	if dias < 0 {
		return 0
	}
	return dias
}
