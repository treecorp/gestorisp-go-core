package worker

import (
	"database/sql"
	"fmt"

	"gestor/internal/entity"
	"gestor/internal/infra/fuso"
	"gestor/internal/lib/iugu"
	"gestor/internal/repositorio"
	"gestor/internal/service/bloqueio"
	"gestor/internal/service/pagamento"
)

// ---------------------------------------------------------------------------
// pagamento.FaturaRepo adapter
// ---------------------------------------------------------------------------

type pagamentoFaturaAdapter struct{}

func (a *pagamentoFaturaAdapter) BuscarFaturaPorToken(db *sql.DB, token string) (*entity.Fatura, error) {
	return repositorio.BuscarFaturaPorToken(db, token)
}

func (a *pagamentoFaturaAdapter) BuscarGatewayToken(q pagamento.Queryer, gatewayID int64) (string, error) {
	var token string
	err := q.QueryRow("SELECT iugu_token FROM sgp_gateway_pagamentos WHERE id = ?", gatewayID).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("buscar gateway token: %w", err)
	}
	return token, nil
}

func (a *pagamentoFaturaAdapter) AtualizarStatusFatura(tx *sql.Tx, faturaID int, valorPago, dataPagto, origem, dataHora, protocoloBaixa string) error {
	dataApenas := dataPagto
	if len(dataPagto) >= 10 {
		dataApenas = dataPagto[:10]
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
		origem_pagamento = ?,
		data_baixa = ?,
		hora_baixa = ?,
		data_hora_pagamento = ?,
		data_hora_baixa = ?,
		protocolo_baixa = ?,
		user_id = 0,
		user_nome = 'Gateway'
		WHERE id = ?`,
		valorPago, dataApenas, origem, dataApenas, horaApenas,
		dataApenas+" 00:00:00", dataHora, protocoloBaixa, faturaID,
	)
	if err != nil {
		return fmt.Errorf("atualizar status fatura: %w", err)
	}
	return nil
}

func (a *pagamentoFaturaAdapter) BuscarSaldoCaixa(tx *sql.Tx, caixaID int) (int, error) {
	var saldo int
	err := tx.QueryRow("SELECT saldo FROM gisp_caixas WHERE id = ?", caixaID).Scan(&saldo)
	if err != nil {
		return 0, fmt.Errorf("buscar saldo caixa %d: %w", caixaID, err)
	}
	return saldo, nil
}

func (a *pagamentoFaturaAdapter) AtualizarSaldoCaixa(tx *sql.Tx, caixaID, novoSaldo int) error {
	_, err := tx.Exec("UPDATE gisp_caixas SET saldo = ? WHERE id = ?", novoSaldo, caixaID)
	if err != nil {
		return fmt.Errorf("atualizar saldo caixa %d: %w", caixaID, err)
	}
	return nil
}

func (a *pagamentoFaturaAdapter) SomarSaldosCaixas(tx *sql.Tx) (int, error) {
	rows, err := tx.Query("SELECT saldo FROM gisp_caixas")
	if err != nil {
		return 0, fmt.Errorf("somar saldos caixas: %w", err)
	}
	defer rows.Close()
	total := 0
	for rows.Next() {
		var s int
		if err := rows.Scan(&s); err != nil {
			return 0, fmt.Errorf("somar saldos caixas: erro scan: %w", err)
		}
		total += s
	}
	return total, rows.Err()
}

func (a *pagamentoFaturaAdapter) InserirFluxoCaixa(tx *sql.Tx, saldoGlobal, saldoAnterior, saldoAtual, valor int, dataHora, token, protocolo, descricao string) error {
	_, err := tx.Exec(`INSERT INTO gisp_fluxos_caixas 
		(saldo_global, saldo_anterior, saldo_atual, valor, operacao, processo, caixa_id, operacao_status, data_hora, token, protocolo, descricao)
		VALUES (?, ?, ?, ?, 'Entrada', 'Pagamento', 1, 'Realizada', ?, ?, ?, ?)`,
		saldoGlobal, saldoAnterior, saldoAtual, valor, dataHora, token, protocolo, descricao,
	)
	if err != nil {
		return fmt.Errorf("inserir fluxo caixa: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// pagamento.ContratoRepo adapter
// ---------------------------------------------------------------------------

type pagamentoContratoAdapter struct{}

func (a *pagamentoContratoAdapter) BuscarContratoPorID(q pagamento.Queryer, contratoID int) (*entity.Contrato, error) {
	return repositorio.BuscarContratoPorID(q, contratoID)
}

func (a *pagamentoContratoAdapter) DesbloquearContrato(tx *sql.Tx, contratoID int) error {
	return repositorio.DesbloquearContrato(tx, contratoID)
}

func (a *pagamentoContratoAdapter) InserirProtocolo(tx *sql.Tx, token, contratoToken string, contratoID int, protocolo, dataHora, descricao, titulo, dadosAntigos, dadosNovos string) error {
	_, err := tx.Exec(`INSERT INTO sgp_clientes_contratos_protocolos 
		(token, contrato_id, contrato_token, protocolo, data_hora, descricao, titulo, dados_antigos, dados_novos, user_id, user_nome)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 'Robot')`,
		token, contratoID, contratoToken, protocolo, dataHora, descricao, titulo, dadosAntigos, dadosNovos,
	)
	if err != nil {
		return fmt.Errorf("inserir protocolo: %w", err)
	}
	return nil
}

func (a *pagamentoContratoAdapter) RemoverRadReplyCorte(tx *sql.Tx, contratoID int) error {
	_, err := tx.Exec("DELETE FROM radreply WHERE sgp_contrato_id = ? AND value = 'pgcorte'", contratoID)
	if err != nil {
		return fmt.Errorf("remover radreply corte: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// pagamento.GatilhoRepo adapter
// ---------------------------------------------------------------------------

type pagamentoGatilhoAdapter struct{}

func (a *pagamentoGatilhoAdapter) InserirGatilhoCompleto(db *sql.DB, iuguFaturaID, accountID, externalRef, status, event, dadosJSON string) error {
	_, err := db.Exec(`INSERT INTO gisp_iugu_gatilhos 
		(id, account_id, external_reference, status, event, dados_json, datetime_received)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE id = id`,
		iuguFaturaID, accountID, externalRef, status, event, dadosJSON,
		fuso.Agora().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return fmt.Errorf("inserir gatilho completo: %w", err)
	}
	return nil
}

func (a *pagamentoGatilhoAdapter) VerificarGatilhoProcessado(db *sql.DB, iuguFaturaID string) (bool, error) {
	var gispExec string
	err := db.QueryRow(`SELECT COALESCE(gisp_exec, '0') FROM gisp_iugu_gatilhos WHERE id = ? AND event = 'invoice.status_changed'`, iuguFaturaID).Scan(&gispExec)
	if err != nil {
		return false, nil
	}
	return gispExec == "1", nil
}

func (a *pagamentoGatilhoAdapter) VerificarGatilhoExternalProcessado(db *sql.DB, iuguFaturaID string) (bool, error) {
	var gispExec string
	err := db.QueryRow(`SELECT gisp_exec FROM gisp_iugu_gatilhos WHERE id = ? AND status = 'externally_paid' AND event = 'invoice.status_changed'`, iuguFaturaID).Scan(&gispExec)
	if err != nil {
		return false, fmt.Errorf("gatilho %s nao encontrado: %w", iuguFaturaID, err)
	}
	return gispExec == "1", nil
}

func (a *pagamentoGatilhoAdapter) InserirGatilho(tx *sql.Tx, iuguFaturaID, statusEsperado string) error {
	return repositorio.InserirGatilho(tx, iuguFaturaID, statusEsperado)
}

func (a *pagamentoGatilhoAdapter) MarcarProcessado(tx *sql.Tx, iuguFaturaID, status, protocolo string) error {
	return repositorio.MarcarProcessado(tx, iuguFaturaID, status, protocolo)
}

func (a *pagamentoGatilhoAdapter) MarcarErroGatilho(db *sql.DB, iuguFaturaID, status, codErro, msg string) error {
	return repositorio.MarcarErroGatilho(db, iuguFaturaID, status, codErro, msg)
}

func (a *pagamentoGatilhoAdapter) SalvarFaturaJSON(tx *sql.Tx, iuguFaturaID string, fatura *iugu.FaturaIugu, dadosJSON string) error {
	_, err := tx.Exec(`INSERT INTO gisp_iugu_faturas_json 
		(id, external_reference, status, dados, total_cents, total_paid_cents, taxes_paid_cents, payment_method)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE status = VALUES(status), dados = VALUES(dados),
		total_cents = VALUES(total_cents), total_paid_cents = VALUES(total_paid_cents),
		taxes_paid_cents = VALUES(taxes_paid_cents), payment_method = VALUES(payment_method)`,
		iuguFaturaID, fatura.ExternalRef, fatura.Status, dadosJSON,
		fatura.TotalCents, fatura.TotalPaidCents, fatura.TaxesPaidCents, fatura.PaymentMethod,
	)
	if err != nil {
		return fmt.Errorf("salvar fatura json: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// pagamento.PopRepo adapter
// ---------------------------------------------------------------------------

type pagamentoPopAdapter struct{}

func (a *pagamentoPopAdapter) BuscarPopsOperacionais(db *sql.DB) ([]entity.Pop, error) {
	return repositorio.BuscarPopsOperacionais(db)
}

// ---------------------------------------------------------------------------
// bloqueio.ContratoRepo adapter
// ---------------------------------------------------------------------------

type bloqueioContratoAdapter struct{}

func (a *bloqueioContratoAdapter) BuscarContratoPorID(q bloqueio.Queryer, contratoID int) (*entity.Contrato, error) {
	return repositorio.BuscarContratoPorID(q, contratoID)
}

// ---------------------------------------------------------------------------
// bloqueio.BloqueioRepo adapter
// ---------------------------------------------------------------------------

type bloqueioBloqueioAdapter struct{}

func (a *bloqueioBloqueioAdapter) BuscarFaturasVencidas(db *sql.DB, diasBloqueio int) ([]entity.Fatura, error) {
	return repositorio.BuscarFaturasVencidas(db, diasBloqueio)
}

func (a *bloqueioBloqueioAdapter) LerDiasBloqueio(db *sql.DB) int {
	return repositorio.LerDiasBloqueio(db)
}

func (a *bloqueioBloqueioAdapter) LerDesbloqueioConfianca(db *sql.DB, contratoID int) (*repositorio.DesbloqueioConfianca, error) {
	return repositorio.LerDesbloqueioConfianca(db, contratoID)
}
