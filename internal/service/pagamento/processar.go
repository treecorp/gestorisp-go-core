package pagamento

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"gestor/internal/entity"
	"gestor/internal/infra/logger"
)

const tag = "service.pagamento"

// ProcessarPagamento é o orquestrador principal do fluxo de pagamento.
// Recebe os dados do webhook Iugu, consulta repositórios, aplica regras
// de negócio e retorna o resultado da baixa.
//
// O fluxo segue esta sequência:
//  1. Registra o gatilho Iugu (idempotente) antes da transação
//  2. Verifica se o gatilho já foi processado (gisp_exec == "1")
//  3. Valida a presença de external_reference
//  4. Roteia para o fluxo adequado:
//     - "externally_paid" → processarExternal
//     - demais          → processarIuguDireto
func ProcessarPagamento(
	repos *Repositorios,
	iuguCli ClienteIugu,
	db *sql.DB,
	instancia entity.Instancia,
	data map[string]string,
	iuguFaturaID string,
	statusEsperado string,
	event string,
) (*ResultadoBaixa, error) {
	externalRef := data["external_reference"]

	if iuguFaturaID == "" {
		return nil, nil
	}

	// INSERT em gisp_iugu_gatilhos (idempotência) — antes da TX
	dadosJSON, _ := json.Marshal(data)
	if err := repos.Gatilho.InserirGatilhoCompleto(
		db, iuguFaturaID, data["account_id"], externalRef,
		data["status"], event, string(dadosJSON),
	); err != nil {
		logger.Aviso(tag, "Instancia %d: erro ao inserir gatilho %s: %v", instancia.ID, iuguFaturaID, err)
	}

	// Verifica se o gatilho já foi processado anteriormente
	processado, err := repos.Gatilho.VerificarGatilhoProcessado(db, iuguFaturaID)
	if err == nil && processado {
		logger.Info(tag, "Instancia %d: gatilho %s ja processado (ignorando)", instancia.ID, iuguFaturaID)
		return nil, nil
	}

	// Se não há external_reference e não é externally_paid, não é possível prosseguir
	if externalRef == "" && statusEsperado != "externally_paid" {
		logger.Aviso(tag, "Instancia %d: external_reference vazio iugu_fatura=%s", instancia.ID, iuguFaturaID)
		return nil, nil
	}

	// Roteia para o fluxo específico
	if statusEsperado == "externally_paid" {
		return processarExternal(repos, iuguCli, db, instancia, data, iuguFaturaID, externalRef, event)
	}

	logger.Info(tag, "Instancia %d: fluxo Iugu direto iugu_fatura=%s ref=%s", instancia.ID, iuguFaturaID, externalRef)
	return processarIuguDireto(repos, iuguCli, db, instancia, data, iuguFaturaID, statusEsperado, externalRef)
}

// processarIuguDireto processa um pagamento via fluxo Iugu direto (paid,
// partially_paid). Busca a fatura pelo token, valida gateway e delega a
// execução da baixa para ExecutarBaixa.
func processarIuguDireto(
	repos *Repositorios,
	iuguCli ClienteIugu,
	db *sql.DB,
	instancia entity.Instancia,
	data map[string]string,
	iuguFaturaID string,
	statusEsperado string,
	externalRef string,
) (*ResultadoBaixa, error) {
	fatura, err := repos.Fatura.BuscarFaturaPorToken(db, externalRef)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: fatura token %s nao encontrada iugu_fatura=%s: %v",
			instancia.ID, externalRef, iuguFaturaID, err)
		_ = repos.Gatilho.MarcarErroGatilho(db, iuguFaturaID, statusEsperado, "Erro 1",
			fmt.Sprintf("Fatura iugu %s nao encontrada", iuguFaturaID))
		return nil, nil
	}
	if fatura == nil {
		logger.Aviso(tag, "Instancia %d: fatura token %s nao encontrada iugu_fatura=%s",
			instancia.ID, externalRef, iuguFaturaID)
		_ = repos.Gatilho.MarcarErroGatilho(db, iuguFaturaID, statusEsperado, "Erro 1",
			fmt.Sprintf("Fatura iugu %s nao encontrada", iuguFaturaID))
		return nil, nil
	}

	if fatura.EstaPaga() {
		logger.Info(tag, "Instancia %d: fatura %d ja estava paga (contrato=%d valor=%s)",
			instancia.ID, fatura.ID, fatura.ContratoID, fatura.Valor)
		_ = repos.Gatilho.MarcarErroGatilho(db, iuguFaturaID, statusEsperado, "Erro 2",
			fmt.Sprintf("Fatura %d ja estava paga", fatura.ID))
		return nil, nil
	}

	logger.Info(tag, "Instancia %d: fatura %d encontrada (contrato=%d valor=%s status_atual=%s)",
		instancia.ID, fatura.ID, fatura.ContratoID, fatura.Valor, fatura.Status)

	if !fatura.GatewayID.Valid {
		logger.Aviso(tag, "Instancia %d: fatura %d sem gateway_id (contrato=%d)",
			instancia.ID, fatura.ID, fatura.ContratoID)
		return nil, nil
	}

	gatewayToken, err := repos.Fatura.BuscarGatewayToken(db, fatura.GatewayID.Int64)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gateway %d nao encontrado para fatura %d: %v",
			instancia.ID, fatura.GatewayID.Int64, fatura.ID, err)
		return nil, nil
	}

	return ExecutarBaixa(repos, iuguCli, db, instancia, data, iuguFaturaID, *fatura, gatewayToken, statusEsperado)
}

// processarExternal processa um pagamento via fluxo externo
// (externally_paid). Verifica se o gatilho já foi processado, busca a
// fatura, valida gateway e delega a execução da baixa para ExecutarBaixa.
func processarExternal(
	repos *Repositorios,
	iuguCli ClienteIugu,
	db *sql.DB,
	instancia entity.Instancia,
	data map[string]string,
	iuguFaturaID string,
	externalRef string,
	event string,
) (*ResultadoBaixa, error) {
	_ = event // mantido para compatibilidade; usado indiretamente via data

	if externalRef == "" {
		logger.Aviso(tag, "Instancia %d: external_reference vazio iugu_fatura=%s", instancia.ID, iuguFaturaID)
		return nil, nil
	}

	// Verifica se o gatilho já foi processado para externally_paid
	processado, err := repos.Gatilho.VerificarGatilhoExternalProcessado(db, iuguFaturaID)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gatilho %s nao encontrado externally_paid: %v", instancia.ID, iuguFaturaID, err)
		return nil, nil
	}
	if processado {
		logger.Info(tag, "Instancia %d: gatilho %s ja processado (externally_paid, ignorando)", instancia.ID, iuguFaturaID)
		return nil, nil
	}

	fatura, err := repos.Fatura.BuscarFaturaPorToken(db, externalRef)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: fatura %s nao encontrada iugu_fatura=%s: %v",
			instancia.ID, externalRef, iuguFaturaID, err)
		_ = repos.Gatilho.MarcarErroGatilho(db, iuguFaturaID, "externally_paid", "Erro 1",
			fmt.Sprintf("Fatura iugu %s nao encontrada", iuguFaturaID))
		return nil, nil
	}
	if fatura == nil {
		logger.Aviso(tag, "Instancia %d: fatura %s nao encontrada iugu_fatura=%s",
			instancia.ID, externalRef, iuguFaturaID)
		_ = repos.Gatilho.MarcarErroGatilho(db, iuguFaturaID, "externally_paid", "Erro 1",
			fmt.Sprintf("Fatura iugu %s nao encontrada", iuguFaturaID))
		return nil, nil
	}

	if fatura.EstaPaga() {
		logger.Info(tag, "Instancia %d: fatura %d ja estava paga (externally_paid, contrato=%d)",
			instancia.ID, fatura.ID, fatura.ContratoID)
		_ = repos.Gatilho.MarcarErroGatilho(db, iuguFaturaID, "externally_paid", "Erro 2",
			fmt.Sprintf("Fatura %d ja estava paga", fatura.ID))
		return nil, nil
	}

	payerName := data["payer_name"]
	logger.Info(tag, "Instancia %d: fatura %d encontrada (contrato=%d valor=%s pagador=%s)",
		instancia.ID, fatura.ID, fatura.ContratoID, fatura.Valor, payerName)

	if !fatura.GatewayID.Valid {
		logger.Aviso(tag, "Instancia %d: fatura %d sem gateway_id", instancia.ID, fatura.ID)
		return nil, nil
	}

	gatewayToken, err := repos.Fatura.BuscarGatewayToken(db, fatura.GatewayID.Int64)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: gateway %d nao encontrado: %v",
			instancia.ID, fatura.GatewayID.Int64, err)
		return nil, nil
	}

	return ExecutarBaixa(repos, iuguCli, db, instancia, data, iuguFaturaID, *fatura, gatewayToken, "externally_paid")
}
