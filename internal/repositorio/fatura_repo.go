package repositorio

import (
	"database/sql"
	"fmt"

	"gestor/internal/entity"
)

// BuscarFaturaPorToken busca uma fatura na tabela sgp_clientes_faturas
// pelo token (external_reference do Iugu). Retorna nil se nao encontrar
// registro (o erro de linhas nao encontradas e convertido para nil).
// Extraído de processar.go: query SELECT da funcao processarIuguDireto.
func BuscarFaturaPorToken(db *sql.DB, token string) (*entity.Fatura, error) {
	var f entity.Fatura
	err := db.QueryRow(`SELECT id, token, valor, contrato_id,
		cliente_token, contrato_token, gateway_id, status
		FROM sgp_clientes_faturas WHERE token = ?`, token).Scan(
		&f.ID, &f.Token, &f.Valor, &f.ContratoID,
		&f.ClienteToken, &f.ContratoToken, &f.GatewayID, &f.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("buscar fatura por token: %w", err)
	}
	return &f, nil
}

// AtualizarStatusFatura atualiza o status da fatura para "Pago",
// registrando valor_pago, data_pagamento, protocolo_baixa e demais
// campos de baixa. Executado dentro de uma transacao.
// Extraído de processar.go: UPDATE na funcao executarBaixa.
func AtualizarStatusFatura(tx *sql.Tx, faturaID int, status string, valorPago string, dataHora string, protocolo string) error {
	dataApenas := dataHora
	if len(dataHora) >= 10 {
		dataApenas = dataHora[:10]
	}
	horaApenas := ""
	if len(dataHora) >= 19 {
		horaApenas = dataHora[11:19]
	}

	_, err := tx.Exec(`UPDATE sgp_clientes_faturas SET
		gateway_status = 'Pago',
		valor_pago = ?,
		data_pagamento = ?,
		bf_paymentToken = NULL,
		status = 'Pago',
		origem_pagamento = '',
		data_baixa = ?,
		hora_baixa = ?,
		data_hora_pagamento = ?,
		data_hora_baixa = ?,
		protocolo_baixa = ?,
		user_id = 0,
		user_nome = 'Gateway'
		WHERE id = ?`,
		valorPago, dataApenas, dataApenas, horaApenas,
		dataApenas+" 00:00:00", dataHora, protocolo, faturaID,
	)
	if err != nil {
		return fmt.Errorf("atualizar status fatura: %w", err)
	}
	return nil
}
