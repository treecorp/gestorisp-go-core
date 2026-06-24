package gateway

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/fuso"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/routeros"
)

type contratoRow struct {
	ID         int
	Token      string
	Status     string
	ClienteID  int
	ClienteToken string
	PopID      int
	PPPoEUser  string
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

func processarPagamento(w http.ResponseWriter, db *sql.DB, data map[string]interface{}, iuguFaturaID string, instancia dominio.Instancia, statusEsperado string) {
	externalRef, _ := data["external_reference"].(string)
	accountID, _ := data["account_id"].(string)

	if len(externalRef) == 9 {
		processarPagamentoJuno(w, db, data, iuguFaturaID, instancia, statusEsperado, accountID, externalRef)
	} else {
		processarPagamentoIuguDireto(w, db, data, iuguFaturaID, instancia, statusEsperado, externalRef)
	}
}

func processarPagamentoIuguDireto(w http.ResponseWriter, db *sql.DB, data map[string]interface{}, iuguFaturaID string, instancia dominio.Instancia, statusEsperado string, externalRef string) {
	if externalRef == "" {
		logger.Aviso(tag, "Instancia %d: external_reference vazio para fatura %s", instancia.ID, iuguFaturaID)
		w.WriteHeader(http.StatusOK)
		return
	}

	var gatilhoStatus, gatilhoEvent, gispExec string
	err := db.QueryRow(`SELECT status, event, gisp_exec FROM gisp_iugu_gatilhos WHERE id = ?`, iuguFaturaID).Scan(&gatilhoStatus, &gatilhoEvent, &gispExec)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gatilho %s nao encontrado: %v", instancia.ID, iuguFaturaID, err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if gispExec == "1" {
		logger.Info(tag, "Instancia %d: gatilho %s ja processado", instancia.ID, iuguFaturaID)
		w.WriteHeader(http.StatusOK)
		return
	}

	var fatura faturaRow
	err = db.QueryRow(`SELECT id, token, valor, contrato_id, cliente_token, contrato_token, gateway_id, status 
		FROM sgp_clientes_faturas WHERE token = ?`, externalRef).Scan(
		&fatura.ID, &fatura.Token, &fatura.Valor, &fatura.ContratoID,
		&fatura.ClienteToken, &fatura.ContratoToken, &fatura.GatewayID, &fatura.Status,
	)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: fatura token %s nao encontrada: %v", instancia.ID, externalRef, err)
		marcarErroGatilho(db, iuguFaturaID, "paid", "Erro 1", fmt.Sprintf("Fatura iugu %s nao encontrada", iuguFaturaID))
		w.WriteHeader(http.StatusOK)
		return
	}
	if fatura.Status == "Pago" {
		logger.Info(tag, "Instancia %d: fatura %d ja estava paga", instancia.ID, fatura.ID)
		marcarErroGatilho(db, iuguFaturaID, "paid", "Erro 2", fmt.Sprintf("Fatura %d ja estava paga", fatura.ID))
		w.WriteHeader(http.StatusOK)
		return
	}

	if !fatura.GatewayID.Valid {
		logger.Aviso(tag, "Instancia %d: fatura %d sem gateway_id", instancia.ID, fatura.ID)
		w.WriteHeader(http.StatusOK)
		return
	}

	var gatewayToken string
	err = db.QueryRow(`SELECT iugu_token FROM sgp_gateway_pagamentos WHERE id = ?`, fatura.GatewayID.Int64).Scan(&gatewayToken)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gateway %d nao encontrado: %v", instancia.ID, fatura.GatewayID.Int64, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	executarBaixa(w, db, data, iuguFaturaID, instancia, fatura, gatewayToken, "", statusEsperado)
}

func processarPagamentoJuno(w http.ResponseWriter, db *sql.DB, data map[string]interface{}, iuguFaturaID string, instancia dominio.Instancia, statusEsperado string, accountID string, externalRef string) {
	var gatilhoStatus, gatilhoEvent, gispExec string
	err := db.QueryRow(`SELECT status, event, gisp_exec FROM gisp_iugu_gatilhos WHERE id = ?`, iuguFaturaID).Scan(&gatilhoStatus, &gatilhoEvent, &gispExec)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gatilho %s nao encontrado: %v", instancia.ID, iuguFaturaID, err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if gispExec == "1" {
		logger.Info(tag, "Instancia %d: gatilho %s ja processado", instancia.ID, iuguFaturaID)
		w.WriteHeader(http.StatusOK)
		return
	}

	var fatura faturaRow
	err = db.QueryRow(`SELECT id, token, valor, contrato_id, cliente_token, contrato_token, gateway_id, status 
		FROM sgp_clientes_faturas WHERE bf_code = ?`, externalRef).Scan(
		&fatura.ID, &fatura.Token, &fatura.Valor, &fatura.ContratoID,
		&fatura.ClienteToken, &fatura.ContratoToken, &fatura.GatewayID, &fatura.Status,
	)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: fatura bf_code %s nao encontrada: %v", instancia.ID, externalRef, err)
		if statusEsperado == "paid" {
			marcarErroGatilho(db, iuguFaturaID, "paid", "Erro 1", fmt.Sprintf("Fatura iugu %s nao encontrada", iuguFaturaID))
		} else {
			marcarErroGatilho(db, iuguFaturaID, "partially_paid", "Erro 1", fmt.Sprintf("Fatura iugu %s nao encontrada", iuguFaturaID))
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	if fatura.Status == "Pago" {
		logger.Info(tag, "Instancia %d: fatura %d ja estava paga", instancia.ID, fatura.ID)
		if statusEsperado == "paid" {
			marcarErroGatilho(db, iuguFaturaID, "paid", "Erro 2", fmt.Sprintf("Fatura %d ja estava paga", fatura.ID))
		} else {
			marcarErroGatilho(db, iuguFaturaID, "partially_paid", "Erro 2", fmt.Sprintf("Fatura %d ja estava paga", fatura.ID))
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	var gatewayToken string
	err = db.QueryRow(`SELECT iugu_token FROM sgp_gateway_pagamentos WHERE iugu_account_id = ?`, accountID).Scan(&gatewayToken)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gateway iugu_account_id %s nao encontrado: %v", instancia.ID, accountID, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	executarBaixa(w, db, data, iuguFaturaID, instancia, fatura, gatewayToken, "Recebido via JUNO atraves da importacao IUGU", statusEsperado)
}

func executarBaixa(w http.ResponseWriter, db *sql.DB, data map[string]interface{}, iuguFaturaID string, instancia dominio.Instancia, fatura faturaRow, gatewayToken string, observacao string, statusEsperado string) {
	logger.Info(tag, "Instancia %d: processando baixa da fatura %d (iugu %s)", instancia.ID, fatura.ID, iuguFaturaID)

	marcarProcessando(db, iuguFaturaID, statusEsperado)

	cliente := NovoCliente(gatewayToken)
	faturaIugu, err := cliente.ConsultarFatura(iuguFaturaID)
	if err != nil {
		logger.Erro(tag, "Instancia %d: erro ao consultar Iugu API fatura %s: %v", instancia.ID, iuguFaturaID, err)
		if statusEsperado == "paid" {
			marcarErroGatilho(db, iuguFaturaID, "paid", "Erro 3", err.Error())
		} else {
			marcarErroGatilho(db, iuguFaturaID, "partially_paid", "Erro 3", err.Error())
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	if faturaIugu.Status != "paid" && faturaIugu.Status != "partially_paid" {
		logger.Aviso(tag, "Instancia %d: fatura %s status inesperado: %s", instancia.ID, iuguFaturaID, faturaIugu.Status)
		if statusEsperado == "paid" {
			marcarErroGatilho(db, iuguFaturaID, "paid", "Erro 4", fmt.Sprintf("Status inesperado: %s", faturaIugu.Status))
		} else {
			marcarErroGatilho(db, iuguFaturaID, "partially_paid", "Erro 4", fmt.Sprintf("Status inesperado: %s", faturaIugu.Status))
		}
		w.WriteHeader(http.StatusOK)
		return
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

	_, err = db.Exec(`UPDATE sgp_clientes_faturas SET 
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
		logger.Erro(tag, "Instancia %d: erro ao atualizar fatura %d: %v", instancia.ID, fatura.ID, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	if statusEsperado == "paid" {
		marcarProcessado(db, iuguFaturaID, "paid", protocolo)
	} else {
		marcarProcessado(db, iuguFaturaID, "partially_paid", protocolo)
	}

	faturaIuguJSON, _ := json.Marshal(faturaIugu)
	_, _ = db.Exec(`INSERT INTO gisp_iugu_faturas_json 
		(id, external_reference, status, dados, total_cents, total_paid_cents, taxes_paid_cents, payment_method)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE status = VALUES(status), dados = VALUES(dados),
		total_cents = VALUES(total_cents), total_paid_cents = VALUES(total_paid_cents),
		taxes_paid_cents = VALUES(taxes_paid_cents), payment_method = VALUES(payment_method)`,
		iuguFaturaID, externalRef(faturaIugu), faturaIugu.Status, string(faturaIuguJSON),
		faturaIugu.TotalCents, faturaIugu.TotalPaidCents, faturaIugu.TaxesPaidCents, faturaIugu.PaymentMethod,
	)

	lancarCaixa(db, fatura, valorPago, dataHora, protocolo)
	criarProtocoloBaixa(db, fatura, valorPago, dataHora, protocolo, observacao)
	desbloquearContrato(db, instancia, fatura.ContratoID, dataHora)

	logger.Sucesso(tag, "Instancia %d: fatura %d baixada com sucesso (protocolo %s)", instancia.ID, fatura.ID, protocolo)
	w.WriteHeader(http.StatusOK)
}

func processarPagamentoExternal(w http.ResponseWriter, db *sql.DB, data map[string]interface{}, iuguFaturaID string, instancia dominio.Instancia) {
	externalRef, _ := data["external_reference"].(string)
	if externalRef == "" {
		logger.Aviso(tag, "Instancia %d: external_reference vazio", instancia.ID)
		w.WriteHeader(http.StatusOK)
		return
	}

	var gispExec string
	err := db.QueryRow(`SELECT gisp_exec FROM gisp_iugu_gatilhos WHERE id = ? AND status = 'externally_paid' AND event = 'invoice.status_changed'`, iuguFaturaID).Scan(&gispExec)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gatilho %s nao encontrado para externally_paid: %v", instancia.ID, iuguFaturaID, err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if gispExec == "1" {
		logger.Info(tag, "Instancia %d: gatilho %s ja processado", instancia.ID, iuguFaturaID)
		w.WriteHeader(http.StatusOK)
		return
	}

	var fatura faturaRow
	err = db.QueryRow(`SELECT id, token, valor, contrato_id, cliente_token, contrato_token, gateway_id, status 
		FROM sgp_clientes_faturas WHERE token = ?`, externalRef).Scan(
		&fatura.ID, &fatura.Token, &fatura.Valor, &fatura.ContratoID,
		&fatura.ClienteToken, &fatura.ContratoToken, &fatura.GatewayID, &fatura.Status,
	)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: fatura %s nao encontrada: %v", instancia.ID, externalRef, err)
		marcarErroGatilho(db, iuguFaturaID, "externally_paid", "Erro 1", fmt.Sprintf("Fatura iugu %s nao encontrada", iuguFaturaID))
		w.WriteHeader(http.StatusOK)
		return
	}
	if fatura.Status == "Pago" {
		marcarErroGatilho(db, iuguFaturaID, "externally_paid", "Erro 2", fmt.Sprintf("Fatura %d ja estava paga", fatura.ID))
		w.WriteHeader(http.StatusOK)
		return
	}

	if !fatura.GatewayID.Valid {
		logger.Aviso(tag, "Instancia %d: fatura %d sem gateway_id", instancia.ID, fatura.ID)
		w.WriteHeader(http.StatusOK)
		return
	}

	var gatewayToken string
	err = db.QueryRow(`SELECT iugu_token FROM sgp_gateway_pagamentos WHERE id = ?`, fatura.GatewayID.Int64).Scan(&gatewayToken)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gateway %d nao encontrado: %v", instancia.ID, fatura.GatewayID.Int64, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	marcarProcessando(db, iuguFaturaID, "externally_paid")

	cliente := NovoCliente(gatewayToken)
	faturaIugu, err := cliente.ConsultarFatura(iuguFaturaID)
	if err != nil {
		logger.Erro(tag, "Instancia %d: erro Iugu API: %v", instancia.ID, err)
		marcarErroGatilho(db, iuguFaturaID, "externally_paid", "Erro 3", err.Error())
		w.WriteHeader(http.StatusOK)
		return
	}

	if faturaIugu.Status != "externally_paid" {
		logger.Aviso(tag, "Instancia %d: status inesperado: %s", instancia.ID, faturaIugu.Status)
		marcarErroGatilho(db, iuguFaturaID, "externally_paid", "Erro 4", fmt.Sprintf("Status inesperado: %s", faturaIugu.Status))
		w.WriteHeader(http.StatusOK)
		return
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

	_, err = db.Exec(`UPDATE sgp_clientes_faturas SET 
		gateway_status = 'Pago',
		valor_pago = ?,
		data_pagamento = ?,
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
		logger.Erro(tag, "Instancia %d: erro ao atualizar fatura %d: %v", instancia.ID, fatura.ID, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	marcarProcessado(db, iuguFaturaID, "externally_paid", protocolo)

	faturaIuguJSON, _ := json.Marshal(faturaIugu)
	_, _ = db.Exec(`INSERT INTO gisp_iugu_faturas_json 
		(id, external_reference, status, dados, total_cents, total_paid_cents, taxes_paid_cents, payment_method)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE status = VALUES(status), dados = VALUES(dados),
		total_cents = VALUES(total_cents), total_paid_cents = VALUES(total_paid_cents),
		taxes_paid_cents = VALUES(taxes_paid_cents), payment_method = VALUES(payment_method)`,
		iuguFaturaID, externalRef, faturaIugu.Status, string(faturaIuguJSON),
		faturaIugu.TotalCents, faturaIugu.TotalPaidCents, faturaIugu.TaxesPaidCents, faturaIugu.PaymentMethod,
	)

	lancarCaixa(db, fatura, valorPago, dataHora, protocolo)
	criarProtocoloBaixa(db, fatura, valorPago, dataHora, protocolo, "")
	desbloquearContrato(db, instancia, fatura.ContratoID, dataHora)

	logger.Sucesso(tag, "Instancia %d: fatura %d baixada (externally_paid, protocolo %s)", instancia.ID, fatura.ID, protocolo)
	w.WriteHeader(http.StatusOK)
}

func externalRef(fatura FaturaIugu) string {
	if fatura.ExternalRef != "" {
		return fatura.ExternalRef
	}
	return ""
}

func marcarProcessando(db *sql.DB, iuguFaturaID string, statusEsperado string) {
	var funcUpdate func(*sql.DB, string, string)
	if statusEsperado == "paid" {
		funcUpdate = func(db *sql.DB, id, _ string) {
			db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec_status = 'Processando' WHERE id = ? AND status = 'paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`, id)
		}
	} else if statusEsperado == "partially_paid" {
		funcUpdate = func(db *sql.DB, id, _ string) {
			db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec_status = 'Processando' WHERE id = ? AND status = 'partially_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`, id)
		}
	} else {
		funcUpdate = func(db *sql.DB, id, _ string) {
			db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec_status = 'Processando' WHERE id = ? AND status = 'externally_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`, id)
		}
	}
	funcUpdate(db, iuguFaturaID, "")
}

func marcarProcessado(db *sql.DB, iuguFaturaID string, status string, protocolo string) {
	if status == "paid" {
		db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = 'Processado', gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			protocolo, fuso.Agora().Format("2006-01-02 15:04:05"), iuguFaturaID)
	} else if status == "partially_paid" {
		db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = 'Processado', gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'partially_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			protocolo, fuso.Agora().Format("2006-01-02 15:04:05"), iuguFaturaID)
	} else {
		db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = 'Processado', gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'externally_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			protocolo, fuso.Agora().Format("2006-01-02 15:04:05"), iuguFaturaID)
	}
}

func marcarErroGatilho(db *sql.DB, iuguFaturaID string, status string, codErro string, msg string) {
	agora := fuso.Agora().Format("2006-01-02 15:04:05")
	if status == "paid" {
		db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = ?, gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			codErro, msg, agora, iuguFaturaID)
	} else if status == "partially_paid" {
		db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = ?, gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'partially_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			codErro, msg, agora, iuguFaturaID)
	} else {
		db.Exec(`UPDATE gisp_iugu_gatilhos SET gisp_exec = '1', gisp_exec_status = ?, gisp_exec_return = ?, datetime_processed = ? WHERE id = ? AND status = 'externally_paid' AND event = 'invoice.status_changed' AND gisp_exec = '0'`,
			codErro, msg, agora, iuguFaturaID)
	}
}

func lancarCaixa(db *sql.DB, fatura faturaRow, valorPago string, dataHora string, protocolo string) {
	var saldoAtual int
	err := db.QueryRow(`SELECT saldo FROM gisp_caixas WHERE id = 1`).Scan(&saldoAtual)
	if err != nil {
		logger.Aviso(tag, "Caixa 1 nao encontrado: %v", err)
		return
	}

	valorNumerico := 0
	fmt.Sscanf(limparNumero(valorPago), "%d", &valorNumerico)
	novoSaldo := saldoAtual + valorNumerico

	_, err = db.Exec(`UPDATE gisp_caixas SET saldo = ? WHERE id = 1`, novoSaldo)
	if err != nil {
		logger.Aviso(tag, "Erro ao atualizar saldo caixa: %v", err)
		return
	}

	caixas, err := db.Query(`SELECT saldo FROM gisp_caixas`)
	if err != nil {
		logger.Aviso(tag, "Erro ao buscar caixas: %v", err)
		return
	}
	defer caixas.Close()

	saldoGlobal := 0
	for caixas.Next() {
		var s int
		caixas.Scan(&s)
		saldoGlobal += s
	}

	descricao := fmt.Sprintf("RECEBIMENTO FAT N %d (B)", fatura.ID)
	token := fmt.Sprintf("%d", rand.Int63())
	seqProtocolo := fmt.Sprintf("%d", gerarProtocolo(100000, 999999))

	_, _ = db.Exec(`INSERT INTO gisp_fluxos_caixas 
		(saldo_global, saldo_anterior, saldo_atual, valor, operacao, processo, caixa_id, operacao_status, data_hora, token, protocolo, descricao)
		VALUES (?, ?, ?, ?, 'Entrada', 'Pagamento', 1, 'Realizada', ?, ?, ?, ?)`,
		saldoGlobal, saldoAtual, novoSaldo, valorNumerico, dataHora, token, seqProtocolo, descricao,
	)
}

func criarProtocoloBaixa(db *sql.DB, fatura faturaRow, valorPago string, dataHora string, protocolo string, observacao string) {
	contrato, err := buscarContrato(db, fatura.ContratoID)
	if err != nil {
		logger.Aviso(tag, "Contrato %d nao encontrado para protocolo: %v", fatura.ContratoID, err)
		return
	}

	agoraShort := fuso.Agora().Format("02/01/2006 15:04")
	descricao := fmt.Sprintf("Fatura n %d valor R$ %s valor recebido R$ %s Contrato n %d baixada em %s",
		fatura.ID, formatarMoeda(fatura.Valor), formatarMoeda(valorPago), contrato.ID, agoraShort)
	if observacao != "" {
		descricao = descricao + ". " + observacao
	}

	dadosAntigos, _ := json.Marshal(map[string]interface{}{"fatura": map[string]interface{}{"id": fatura.ID, "status": fatura.Status}})
	dadosNovos, _ := json.Marshal(map[string]interface{}{"fatura": map[string]interface{}{"id": fatura.ID, "status": "Pago"}})

	token := fmt.Sprintf("tok_%d", rand.Int63())[:32]

	_, _ = db.Exec(`INSERT INTO sgp_clientes_contratos_protocolos 
		(token, contrato_id, contrato_token, protocolo, data_hora, descricao, titulo, dados_antigos, dados_novos, user_id, user_nome)
		VALUES (?, ?, ?, ?, ?, ?, 'Baixa de fatura', ?, ?, 0, 'Robot')`,
		token, contrato.ID, contrato.Token, protocolo, dataHora, descricao,
		string(dadosAntigos), string(dadosNovos),
	)
}

func desbloquearContrato(db *sql.DB, instancia dominio.Instancia, contratoID int, dataHora string) {
	contrato, err := buscarContrato(db, contratoID)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: contrato %d nao encontrado: %v", instancia.ID, contratoID, err)
		return
	}

	if contrato.Status != "Bloqueado" {
		logger.Info(tag, "Instancia %d: contrato %d nao esta bloqueado (status=%s)", instancia.ID, contratoID, contrato.Status)
		return
	}

	logger.Info(tag, "Instancia %d: desbloqueando contrato %d", instancia.ID, contratoID)

	_, err = db.Exec(`UPDATE sgp_clientes_contratos SET status = 'Ativo' WHERE id = ?`, contratoID)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: erro ao desbloquear contrato %d: %v", instancia.ID, contratoID, err)
		return
	}

	_, _ = db.Exec(`DELETE FROM radreply WHERE sgp_contrato_id = ? AND value = 'pgcorte'`, contratoID)

	agoraShort := fuso.Agora().Format("02/01/2006 15:04")
	descricao := fmt.Sprintf("Contrato n %d desbloqueado em %s", contratoID, agoraShort)
	dadosAntigos, _ := json.Marshal(map[string]interface{}{"contrato": map[string]interface{}{"id": contratoID, "status": "Bloqueado"}})
	dadosNovos, _ := json.Marshal(map[string]interface{}{"contrato": map[string]interface{}{"id": contratoID, "status": "Ativo"}})
	token := fmt.Sprintf("tok_%d", rand.Int63())[:32]
	bloqProtocolo := fmt.Sprintf("%d", gerarProtocolo(400000, 499999))

	_, _ = db.Exec(`INSERT INTO sgp_clientes_contratos_protocolos 
		(token, contrato_id, contrato_token, protocolo, data_hora, descricao, titulo, dados_antigos, dados_novos, user_id, user_nome)
		VALUES (?, ?, ?, ?, ?, ?, 'Desbloqueio de contrato', ?, ?, 0, 'Robot')`,
		token, contrato.ID, contrato.Token, bloqProtocolo, dataHora, descricao,
		string(dadosAntigos), string(dadosNovos),
	)

	pops, err := banco.BuscarPopsOperacionais(db)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: erro ao buscar POPs: %v", instancia.ID, err)
		return
	}

	mapaPops := make(map[int]dominio.Pop)
	for _, p := range pops {
		mapaPops[p.ID] = p
	}

	pop, ok := mapaPops[contrato.PopID]
	if !ok {
		logger.Aviso(tag, "Instancia %d: POP %d nao encontrado para contrato %d", instancia.ID, contrato.PopID, contratoID)
		return
	}

	conn, err := routeros.Conectar(routeros.DadosConexao{
		IPv4: pop.IPv4,
		Port: pop.APIPort,
		User: pop.User,
		Pass: pop.Pass,
	})
	if err != nil {
		logger.Aviso(tag, "Instancia %d: POP %d (%s) inacessivel: %v", instancia.ID, pop.ID, pop.IPv4, err)
		return
	}
	defer conn.Close()

	ativo, sessionID, err := routeros.VerificarUsuarioAtivo(conn, contrato.PPPoEUser)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: erro ao consultar RB para %s: %v", instancia.ID, contrato.PPPoEUser, err)
		return
	}

	if !ativo {
		logger.Info(tag, "Instancia %d: %s ja desconectado da RB", instancia.ID, contrato.PPPoEUser)
		return
	}

	if err := routeros.DesconectarUsuario(conn, sessionID); err != nil {
		logger.Aviso(tag, "Instancia %d: erro ao desconectar %s da RB: %v", instancia.ID, contrato.PPPoEUser, err)
		return
	}

	logger.Sucesso(tag, "Instancia %d: %s desconectado da RB (POP %d)", instancia.ID, contrato.PPPoEUser, pop.ID)
}

func buscarContrato(db *sql.DB, contratoID int) (*contratoRow, error) {
	var c contratoRow
	err := db.QueryRow(`SELECT id, token, status, cliente_id, cliente_token, pop_id, pppoe_user 
		FROM sgp_clientes_contratos WHERE id = ?`, contratoID).Scan(
		&c.ID, &c.Token, &c.Status, &c.ClienteID, &c.ClienteToken, &c.PopID, &c.PPPoEUser,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
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
