package pagamento

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/fuso"
	"gestor/internal/infra/logger"
)

var codigosOrigem = map[string]string{
	"iugu_pix":            "5",
	"iugu_pix_test":       "5",
	"iugu_credit_card":    "4",
	"iugu_bank_slip":      "7",
	"iugu_bank_slip_test": "7",
}

type contratoRow struct {
	ID           int
	Token        string
	Status       string
	ClienteID    int
	ClienteNome  string
	ClienteToken string
	PopID        int
	PPPoEUser    string
}

type faturaRow struct {
	ID            int
	Token         string
	Valor         string
	ContratoID    int
	ClienteToken  string
	ContratoToken string
	GatewayID     sql.NullInt64
	Status        string
}

type ResultadoBaixa struct {
	ContratoID         int
	ClienteNome        string
	PPPoEUser          string
	PopIPv4            string
	PopPort            string
	PopUser            string
	PopPass            string
	PrecisaDesconectar bool
}

const tag = "pagamento"

func ProcessarPagamento(db *sql.DB, instancia dominio.Instancia, data map[string]string, iuguFaturaID string, statusEsperado string) (*ResultadoBaixa, error) {
	externalRef := data["external_reference"]

	if iuguFaturaID == "" {
		return nil, nil
	}

	var gispExec string
	err := db.QueryRow(`SELECT COALESCE(gisp_exec, '0') FROM gisp_iugu_gatilhos WHERE id = ?`, iuguFaturaID).Scan(&gispExec)
	if err == nil && gispExec == "1" {
		logger.Info(tag, "Instancia %d: gatilho %s ja processado (ignorando)", instancia.ID, iuguFaturaID)
		return nil, nil
	}

	if externalRef == "" && statusEsperado != "externally_paid" {
		logger.Aviso(tag, "Instancia %d: external_reference vazio iugu_fatura=%s", instancia.ID, iuguFaturaID)
		return nil, nil
	}

	if statusEsperado == "externally_paid" {
		return processarExternal(db, instancia, data, iuguFaturaID, externalRef)
	}

	if len(externalRef) == 9 {
		accountID := data["account_id"]
		logger.Info(tag, "Instancia %d: fluxo Juno via Iugu iugu_fatura=%s ref=%s", instancia.ID, iuguFaturaID, externalRef)
		return processarJuno(db, instancia, data, iuguFaturaID, statusEsperado, accountID, externalRef)
	}

	logger.Info(tag, "Instancia %d: fluxo Iugu direto iugu_fatura=%s ref=%s", instancia.ID, iuguFaturaID, externalRef)
	return processarIuguDireto(db, instancia, data, iuguFaturaID, statusEsperado, externalRef)
}

func processarIuguDireto(db *sql.DB, instancia dominio.Instancia, data map[string]string, iuguFaturaID string, statusEsperado string, externalRef string) (*ResultadoBaixa, error) {
	var fatura faturaRow
	err := db.QueryRow(`SELECT id, token, valor, contrato_id, cliente_token, contrato_token, gateway_id, status 
		FROM sgp_clientes_faturas WHERE token = ?`, externalRef).Scan(
		&fatura.ID, &fatura.Token, &fatura.Valor, &fatura.ContratoID,
		&fatura.ClienteToken, &fatura.ContratoToken, &fatura.GatewayID, &fatura.Status,
	)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: fatura token %s nao encontrada iugu_fatura=%s: %v", instancia.ID, externalRef, iuguFaturaID, err)
		marcarErroGatilho(db, iuguFaturaID, statusEsperado, "Erro 1", fmt.Sprintf("Fatura iugu %s nao encontrada", iuguFaturaID))
		return nil, nil
	}
	if fatura.Status == "Pago" {
		logger.Info(tag, "Instancia %d: fatura %d ja estava paga (contrato=%d valor=%s)", instancia.ID, fatura.ID, fatura.ContratoID, fatura.Valor)
		marcarErroGatilho(db, iuguFaturaID, statusEsperado, "Erro 2", fmt.Sprintf("Fatura %d ja estava paga", fatura.ID))
		return nil, nil
	}

	logger.Info(tag, "Instancia %d: fatura %d encontrada (contrato=%d valor=%s status_atual=%s)", instancia.ID, fatura.ID, fatura.ContratoID, fatura.Valor, fatura.Status)

	if !fatura.GatewayID.Valid {
		logger.Aviso(tag, "Instancia %d: fatura %d sem gateway_id (contrato=%d)", instancia.ID, fatura.ID, fatura.ContratoID)
		return nil, nil
	}

	var gatewayToken string
	err = db.QueryRow(`SELECT iugu_token FROM sgp_gateway_pagamentos WHERE id = ?`, fatura.GatewayID.Int64).Scan(&gatewayToken)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gateway %d nao encontrado para fatura %d: %v", instancia.ID, fatura.GatewayID.Int64, fatura.ID, err)
		return nil, nil
	}

	return executarBaixa(db, instancia, data, iuguFaturaID, fatura, gatewayToken, "", statusEsperado)
}

func processarJuno(db *sql.DB, instancia dominio.Instancia, data map[string]string, iuguFaturaID string, statusEsperado string, accountID string, externalRef string) (*ResultadoBaixa, error) {
	var fatura faturaRow
	err := db.QueryRow(`SELECT id, token, valor, contrato_id, cliente_token, contrato_token, gateway_id, status 
		FROM sgp_clientes_faturas WHERE bf_code = ?`, externalRef).Scan(
		&fatura.ID, &fatura.Token, &fatura.Valor, &fatura.ContratoID,
		&fatura.ClienteToken, &fatura.ContratoToken, &fatura.GatewayID, &fatura.Status,
	)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: fatura bf_code %s nao encontrada iugu_fatura=%s: %v", instancia.ID, externalRef, iuguFaturaID, err)
		marcarErroGatilho(db, iuguFaturaID, statusEsperado, "Erro 1", fmt.Sprintf("Fatura iugu %s nao encontrada", iuguFaturaID))
		return nil, nil
	}
	if fatura.Status == "Pago" {
		logger.Info(tag, "Instancia %d: fatura %d ja estava paga (Juno, contrato=%d valor=%s)", instancia.ID, fatura.ID, fatura.ContratoID, fatura.Valor)
		marcarErroGatilho(db, iuguFaturaID, statusEsperado, "Erro 2", fmt.Sprintf("Fatura %d ja estava paga", fatura.ID))
		return nil, nil
	}

	logger.Info(tag, "Instancia %d: fatura %d encontrada via bf_code (contrato=%d valor=%s)", instancia.ID, fatura.ID, fatura.ContratoID, fatura.Valor)

	var gatewayToken string
	err = db.QueryRow(`SELECT iugu_token FROM sgp_gateway_pagamentos WHERE iugu_account_id = ?`, accountID).Scan(&gatewayToken)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gateway iugu_account_id %s nao encontrado: %v", instancia.ID, accountID, err)
		return nil, nil
	}

	return executarBaixa(db, instancia, data, iuguFaturaID, fatura, gatewayToken, "Recebido via JUNO atraves da importacao IUGU", statusEsperado)
}

func processarExternal(db *sql.DB, instancia dominio.Instancia, data map[string]string, iuguFaturaID string, externalRef string) (*ResultadoBaixa, error) {
	payerName := data["payer_name"]
	if externalRef == "" {
		logger.Aviso(tag, "Instancia %d: external_reference vazio iugu_fatura=%s", instancia.ID, iuguFaturaID)
		return nil, nil
	}

	var gispExec string
	err := db.QueryRow(`SELECT gisp_exec FROM gisp_iugu_gatilhos WHERE id = ? AND status = 'externally_paid' AND event = 'invoice.status_changed'`, iuguFaturaID).Scan(&gispExec)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gatilho %s nao encontrado externally_paid: %v", instancia.ID, iuguFaturaID, err)
		return nil, nil
	}
	if gispExec == "1" {
		logger.Info(tag, "Instancia %d: gatilho %s ja processado (externally_paid, ignorando)", instancia.ID, iuguFaturaID)
		return nil, nil
	}

	var fatura faturaRow
	err = db.QueryRow(`SELECT id, token, valor, contrato_id, cliente_token, contrato_token, gateway_id, status 
		FROM sgp_clientes_faturas WHERE token = ?`, externalRef).Scan(
		&fatura.ID, &fatura.Token, &fatura.Valor, &fatura.ContratoID,
		&fatura.ClienteToken, &fatura.ContratoToken, &fatura.GatewayID, &fatura.Status,
	)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: fatura %s nao encontrada iugu_fatura=%s: %v", instancia.ID, externalRef, iuguFaturaID, err)
		marcarErroGatilho(db, iuguFaturaID, "externally_paid", "Erro 1", fmt.Sprintf("Fatura iugu %s nao encontrada", iuguFaturaID))
		return nil, nil
	}
	if fatura.Status == "Pago" {
		logger.Info(tag, "Instancia %d: fatura %d ja estava paga (externally_paid, contrato=%d)", instancia.ID, fatura.ID, fatura.ContratoID)
		marcarErroGatilho(db, iuguFaturaID, "externally_paid", "Erro 2", fmt.Sprintf("Fatura %d ja estava paga", fatura.ID))
		return nil, nil
	}

	logger.Info(tag, "Instancia %d: fatura %d encontrada (contrato=%d valor=%s pagador=%s)", instancia.ID, fatura.ID, fatura.ContratoID, fatura.Valor, payerName)

	if !fatura.GatewayID.Valid {
		logger.Aviso(tag, "Instancia %d: fatura %d sem gateway_id", instancia.ID, fatura.ID)
		return nil, nil
	}

	var gatewayToken string
	err = db.QueryRow(`SELECT iugu_token FROM sgp_gateway_pagamentos WHERE id = ?`, fatura.GatewayID.Int64).Scan(&gatewayToken)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gateway %d nao encontrado: %v", instancia.ID, fatura.GatewayID.Int64, err)
		return nil, nil
	}

	return executarBaixa(db, instancia, data, iuguFaturaID, fatura, gatewayToken, "", "externally_paid")
}

func executarBaixa(db *sql.DB, instancia dominio.Instancia, data map[string]string, iuguFaturaID string, fatura faturaRow, gatewayToken string, observacao string, statusEsperado string) (*ResultadoBaixa, error) {
	payerName := data["payer_name"]
	logger.Info(tag, "Instancia %d: baixando fatura %d (contrato=%d valor=%s pagador=%s)", instancia.ID, fatura.ID, fatura.ContratoID, fatura.Valor, payerName)

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("erro ao iniciar transacao: %w", err)
	}
	defer tx.Rollback()

	marcarProcessando(tx, iuguFaturaID, statusEsperado)

	cliente := NovoCliente(gatewayToken)
	faturaIugu, err := cliente.ConsultarFatura(iuguFaturaID)
	if err != nil {
		logger.Erro(tag, "Instancia %d: erro Iugu API fatura %s: %v", instancia.ID, iuguFaturaID, err)
		return nil, fmt.Errorf("erro Iugu API fatura %s: %w", iuguFaturaID, err)
	}

	if statusEsperado == "externally_paid" && faturaIugu.Status != "externally_paid" {
		logger.Aviso(tag, "Instancia %d: fatura %s status inesperado para externally_paid: %s", instancia.ID, iuguFaturaID, faturaIugu.Status)
		return nil, nil
	}
	if statusEsperado != "externally_paid" && faturaIugu.Status != "paid" && faturaIugu.Status != "partially_paid" {
		logger.Aviso(tag, "Instancia %d: fatura %s status inesperado: %s", instancia.ID, iuguFaturaID, faturaIugu.Status)
		return nil, nil
	}

	origem := origemPagamento(faturaIugu.PaymentMethod)
	agora := fuso.Agora()
	dataPagto := faturaIugu.PaidAt
	if len(dataPagto) >= 10 {
		dataPagto = dataPagto[:10]
	}

	protocolo := fmt.Sprintf("%d", gerarProtocolo(100000, 999999))
	dataHora := agora.Format("2006-01-02 15:04:05")
	dataAtual := agora.Format("2006-01-02")
	horaAtual := agora.Format("15:04:05")
	valorPago := fmt.Sprintf("%d", faturaIugu.TotalPaidCents)

	_, err = tx.Exec(`UPDATE sgp_clientes_faturas SET 
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
		valorPago, dataPagto, origem, dataAtual, horaAtual,
		dataPagto+" 00:00:00", dataHora, protocolo, fatura.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao atualizar fatura %d: %w", fatura.ID, err)
	}

	marcarProcessado(tx, iuguFaturaID, statusEsperado, protocolo)

	faturaIuguJSON, _ := json.Marshal(faturaIugu)
	_, _ = tx.Exec(`INSERT INTO gisp_iugu_faturas_json 
		(id, external_reference, status, dados, total_cents, total_paid_cents, taxes_paid_cents, payment_method)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE status = VALUES(status), dados = VALUES(dados),
		total_cents = VALUES(total_cents), total_paid_cents = VALUES(total_paid_cents),
		taxes_paid_cents = VALUES(taxes_paid_cents), payment_method = VALUES(payment_method)`,
		iuguFaturaID, externalRef(faturaIugu), faturaIugu.Status, string(faturaIuguJSON),
		faturaIugu.TotalCents, faturaIugu.TotalPaidCents, faturaIugu.TaxesPaidCents, faturaIugu.PaymentMethod,
	)

	lancarCaixa(tx, fatura, valorPago, dataHora, protocolo)
	contrato := criarProtocoloBaixa(tx, fatura, valorPago, dataHora, protocolo, observacao)
	resultado := desbloquearContratoDB(tx, db, instancia, fatura.ContratoID, dataHora, contrato)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("erro ao commitar transacao: %w", err)
	}

	logger.Sucesso(tag, "Instancia %d: fatura %d baixada (contrato=%d cliente=%s protocolo=%s)", instancia.ID, fatura.ID, fatura.ContratoID, payerName, protocolo)
	return resultado, nil
}

func marcarProcessando(tx *sql.Tx, iuguFaturaID string, statusEsperado string) {
	switch statusEsperado {
	case "paid":
		tx.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec_status = 'Processando' WHERE id = ? AND status = 'paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`, iuguFaturaID)
	case "partially_paid":
		tx.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec_status = 'Processando' WHERE id = ? AND status = 'partially_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`, iuguFaturaID)
	default:
		tx.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec_status = 'Processando' WHERE id = ? AND status = 'externally_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`, iuguFaturaID)
	}
}

func marcarProcessado(tx *sql.Tx, iuguFaturaID string, status string, protocolo string) {
	agora := fuso.Agora().Format("2006-01-02 15:04:05")
	switch status {
	case "paid":
		tx.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = 'Processado', gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			protocolo, agora, iuguFaturaID)
	case "partially_paid":
		tx.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = 'Processado', gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'partially_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			protocolo, agora, iuguFaturaID)
	default:
		tx.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = 'Processado', gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'externally_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			protocolo, agora, iuguFaturaID)
	}
}

func marcarErroGatilho(db *sql.DB, iuguFaturaID string, status string, codErro string, msg string) {
	agora := fuso.Agora().Format("2006-01-02 15:04:05")
	switch status {
	case "paid":
		db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = ?, gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			codErro, msg, agora, iuguFaturaID)
	case "partially_paid":
		db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = ?, gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'partially_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			codErro, msg, agora, iuguFaturaID)
	default:
		db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = ?, gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'externally_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			codErro, msg, agora, iuguFaturaID)
	}
}

func lancarCaixa(tx *sql.Tx, fatura faturaRow, valorPago string, dataHora string, protocolo string) {
	var saldoAtual int
	err := tx.QueryRow(`SELECT saldo FROM gisp_caixas WHERE id = 1`).Scan(&saldoAtual)
	if err != nil {
		logger.Aviso(tag, "Caixa 1 nao encontrado para fatura %d: %v", fatura.ID, err)
		return
	}

	valorNumerico := 0
	fmt.Sscanf(limparNumero(valorPago), "%d", &valorNumerico)
	novoSaldo := saldoAtual + valorNumerico

	_, err = tx.Exec(`UPDATE gisp_caixas SET saldo = ? WHERE id = 1`, novoSaldo)
	if err != nil {
		logger.Aviso(tag, "Erro ao atualizar saldo caixa fatura %d: %v", fatura.ID, err)
		return
	}

	saldoGlobal := 0
	rows, err := tx.Query(`SELECT saldo FROM gisp_caixas`)
	if err != nil {
		logger.Aviso(tag, "Erro ao buscar caixas fatura %d: %v", fatura.ID, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var s int
		rows.Scan(&s)
		saldoGlobal += s
	}

	descricao := fmt.Sprintf("RECEBIMENTO FAT N %d (B)", fatura.ID)
	token := fmt.Sprintf("%d", rand.Int63())
	seqProtocolo := fmt.Sprintf("%d", gerarProtocolo(100000, 999999))

	_, _ = tx.Exec(`INSERT INTO gisp_fluxos_caixas 
		(saldo_global, saldo_anterior, saldo_atual, valor, operacao, processo, caixa_id, operacao_status, data_hora, token, protocolo, descricao)
		VALUES (?, ?, ?, ?, 'Entrada', 'Pagamento', 1, 'Realizada', ?, ?, ?, ?)`,
		saldoGlobal, saldoAtual, novoSaldo, valorNumerico, dataHora, token, seqProtocolo, descricao,
	)

	logger.Info(tag, "Caixa fatura %d: saldo %d -> %d (global=%d)", fatura.ID, saldoAtual, novoSaldo, saldoGlobal)
}

func criarProtocoloBaixa(tx *sql.Tx, fatura faturaRow, valorPago string, dataHora string, protocolo string, observacao string) *contratoRow {
	contrato, err := buscarContrato(tx, fatura.ContratoID)
	if err != nil {
		logger.Aviso(tag, "Contrato %d nao encontrado protocolo fatura %d: %v", fatura.ContratoID, fatura.ID, err)
		return nil
	}

	agoraShort := fuso.Agora().Format("02/01/2006 15:04")
	descricao := fmt.Sprintf("Fatura n %d valor R$ %s valor recebido R$ %s Contrato n %d (%s) baixada em %s",
		fatura.ID, formatarMoeda(fatura.Valor), formatarMoeda(valorPago), contrato.ID, contrato.ClienteNome, agoraShort)
	if observacao != "" {
		descricao = descricao + ". " + observacao
	}

	dadosAntigos, _ := json.Marshal(map[string]interface{}{"fatura": map[string]interface{}{"id": fatura.ID, "status": fatura.Status}})
	dadosNovos, _ := json.Marshal(map[string]interface{}{"fatura": map[string]interface{}{"id": fatura.ID, "status": "Pago"}})

	token := fmt.Sprintf("tok_%d", rand.Int63())[:32]

	_, _ = tx.Exec(`INSERT INTO sgp_clientes_contratos_protocolos 
		(token, contrato_id, contrato_token, protocolo, data_hora, descricao, titulo, dados_antigos, dados_novos, user_id, user_nome)
		VALUES (?, ?, ?, ?, ?, ?, 'Baixa de fatura', ?, ?, 0, 'Robot')`,
		token, contrato.ID, contrato.Token, protocolo, dataHora, descricao,
		string(dadosAntigos), string(dadosNovos),
	)

	logger.Info(tag, "Protocolo baixa %s gerado: fatura=%d contrato=%d cliente=%s", protocolo, fatura.ID, contrato.ID, contrato.ClienteNome)
	return contrato
}

func desbloquearContratoDB(tx *sql.Tx, db *sql.DB, instancia dominio.Instancia, contratoID int, dataHora string, contrato *contratoRow) *ResultadoBaixa {
	if contrato == nil {
		var err error
		contrato, err = buscarContrato(tx, contratoID)
		if err != nil {
			logger.Aviso(tag, "Instancia %d: contrato %d nao encontrado desbloqueio: %v", instancia.ID, contratoID, err)
			return nil
		}
	}

	if contrato.Status != "Bloqueado" {
		logger.Info(tag, "Instancia %d: contrato %d (%s) nao esta bloqueado (status=%s)", instancia.ID, contratoID, contrato.ClienteNome, contrato.Status)
		return nil
	}

	logger.Info(tag, "Instancia %d: desbloqueando contrato %d (%s pppoe=%s)", instancia.ID, contratoID, contrato.ClienteNome, contrato.PPPoEUser)

	_, err := tx.Exec(`UPDATE sgp_clientes_contratos SET status = 'Ativo' WHERE id = ?`, contratoID)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: erro ao desbloquear contrato %d: %v", instancia.ID, contratoID, err)
		return nil
	}

	_, _ = tx.Exec(`DELETE FROM radreply WHERE sgp_contrato_id = ? AND value = 'pgcorte'`, contratoID)

	agoraShort := fuso.Agora().Format("02/01/2006 15:04")
	descricao := fmt.Sprintf("Contrato n %d (%s) desbloqueado em %s", contratoID, contrato.ClienteNome, agoraShort)
	dadosAntigos, _ := json.Marshal(map[string]interface{}{"contrato": map[string]interface{}{"id": contratoID, "status": "Bloqueado"}})
	dadosNovos, _ := json.Marshal(map[string]interface{}{"contrato": map[string]interface{}{"id": contratoID, "status": "Ativo"}})
	token := fmt.Sprintf("tok_%d", rand.Int63())[:32]
	bloqProtocolo := fmt.Sprintf("%d", gerarProtocolo(400000, 499999))

	_, _ = tx.Exec(`INSERT INTO sgp_clientes_contratos_protocolos 
		(token, contrato_id, contrato_token, protocolo, data_hora, descricao, titulo, dados_antigos, dados_novos, user_id, user_nome)
		VALUES (?, ?, ?, ?, ?, ?, 'Desbloqueio de contrato', ?, ?, 0, 'Robot')`,
		token, contrato.ID, contrato.Token, bloqProtocolo, dataHora, descricao,
		string(dadosAntigos), string(dadosNovos),
	)

	pops, err := banco.BuscarPopsOperacionais(db)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: erro ao buscar POPs: %v", instancia.ID, err)
		return nil
	}

	mapaPops := make(map[int]dominio.Pop)
	for _, p := range pops {
		mapaPops[p.ID] = p
	}

	pop, ok := mapaPops[contrato.PopID]
	if !ok {
		logger.Aviso(tag, "Instancia %d: POP %d nao encontrado contrato %d (%s)", instancia.ID, contrato.PopID, contratoID, contrato.ClienteNome)
		return nil
	}

	return &ResultadoBaixa{
		ContratoID:         contratoID,
		ClienteNome:        contrato.ClienteNome,
		PPPoEUser:          contrato.PPPoEUser,
		PopIPv4:            pop.IPv4,
		PopPort:            pop.APIPort,
		PopUser:            pop.User,
		PopPass:            pop.Pass,
		PrecisaDesconectar: true,
	}
}

func buscarContrato(q queryer, contratoID int) (*contratoRow, error) {
	var c contratoRow
	err := q.QueryRow(`SELECT c.id, c.token, c.status, c.cliente_id, 
		COALESCE(cli.pf_nome, cli.pj_razao_social, 'N/D') AS cliente_nome,
		c.cliente_token, c.pop_id, c.pppoe_user 
		FROM sgp_clientes_contratos c
		LEFT JOIN sgp_clientes_new cli ON cli.id = c.cliente_id
		WHERE c.id = ?`, contratoID).Scan(
		&c.ID, &c.Token, &c.Status, &c.ClienteID,
		&c.ClienteNome, &c.ClienteToken, &c.PopID, &c.PPPoEUser,
	)
	if err != nil {
		return nil, err
	}
	logger.Info(tag, "Contrato %d: cliente=%s pppoe=%s pop=%d", c.ID, c.ClienteNome, c.PPPoEUser, c.PopID)
	return &c, nil
}

type queryer interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

func externalRef(fatura FaturaIugu) string {
	if fatura.ExternalRef != "" {
		return fatura.ExternalRef
	}
	return ""
}

func origemPagamento(metodo string) string {
	if cod, ok := codigosOrigem[metodo]; ok {
		return cod
	}
	return "7"
}

func gerarProtocolo(min, max int) int {
	return min + (idCounter() % (max - min + 1))
}

var counter int

func idCounter() int {
	counter++
	return counter
}

func formatarMoeda(valor string) string {
	limpo := limparNumero(valor)
	if len(limpo) <= 2 {
		return "0," + limpo
	}
	reais := limpo[:len(limpo)-2]
	centavos := limpo[len(limpo)-2:]
	return reais + "," + centavos
}

func limparNumero(valor string) string {
	r := strings.NewReplacer(".", "", ",", "", "R$", "", " ", "")
	return r.Replace(valor)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
