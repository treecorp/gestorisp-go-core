package pagamento

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"

	"gestor/internal/entity"
	"gestor/internal/helpers"
	"gestor/internal/infra/fuso"
	"gestor/internal/infra/logger"
	"gestor/internal/lib/iugu"
)

// ResultadoBaixa contém os dados retornados após o processamento da baixa
// de uma fatura. Inclui informações para desconexão do cliente (POP) quando
// o contrato é desbloqueado.
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

const tagBaixa = "pagamento"

// ExecutarBaixa executa a lógica completa de baixa contábil de uma fatura
// já confirmada como paga pela API Iugu. Orquestra a atualização da fatura,
// registro do gatilho, lançamento em caixa, geração de protocolo e
// desbloqueio do contrato, tudo dentro de uma única transação.
func ExecutarBaixa(
	repos *Repositorios,
	iuguCli ClienteIugu,
	db *sql.DB,
	instancia entity.Instancia,
	data map[string]string,
	iuguFaturaID string,
	fatura entity.Fatura,
	gatewayToken string,
	statusEsperado string,
) (*ResultadoBaixa, error) {
	payerName := data["payer_name"]
	logger.Info(tagBaixa, "Instancia %d: baixando fatura %d (contrato=%d valor=%s pagador=%s)",
		instancia.ID, fatura.ID, fatura.ContratoID, fatura.Valor, payerName)

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("erro ao iniciar transacao: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Marca o gatilho como "Processando" dentro da transação
	if err := repos.Gatilho.InserirGatilho(tx, iuguFaturaID, statusEsperado); err != nil {
		return nil, fmt.Errorf("erro ao marcar processando gatilho %s: %w", iuguFaturaID, err)
	}

	// Consulta a fatura na API Iugu usando o token do gateway
	faturaIugu, err := iuguCli.ConsultarFatura(iuguFaturaID)
	if err != nil {
		logger.Erro(tagBaixa, "Instancia %d: erro Iugu API fatura %s: %v", instancia.ID, iuguFaturaID, err)
		return nil, fmt.Errorf("erro Iugu API fatura %s: %w", iuguFaturaID, err)
	}

	// Valida o status retornado pela Iugu conforme o fluxo esperado
	if err := validarStatusIugu(faturaIugu, statusEsperado); err != nil {
		logger.Aviso(tagBaixa, "Instancia %d: fatura %s status invalido: %v", instancia.ID, iuguFaturaID, err)
		return nil, nil
	}

	origem := OrigemPagamento(faturaIugu.PaymentMethod)
	agora := fuso.Agora()
	dataPagto := faturaIugu.PaidAt
	if len(dataPagto) >= 10 {
		dataPagto = dataPagto[:10]
	}

	protocoloBaixa := fmt.Sprintf("%d", helpers.GerarProtocolo(100000, 999999)) // protocolo_baixa da fatura
	protocolo := fmt.Sprintf("%d", helpers.GerarProtocolo(300000, 399999))      // protocolo do registro de baixa
	dataHora := agora.Format("2006-01-02 15:04:05")
	valorPago := fmt.Sprintf("%d", faturaIugu.TotalPaidCents)

	// Atualiza o status da fatura para Pago
	if err := repos.Fatura.AtualizarStatusFatura(tx, fatura.ID, valorPago, dataPagto, origem, dataHora, protocoloBaixa); err != nil {
		return nil, fmt.Errorf("erro ao atualizar fatura %d: %w", fatura.ID, err)
	}

	// Marca o gatilho como processado
	if err := repos.Gatilho.MarcarProcessado(tx, iuguFaturaID, statusEsperado, protocolo); err != nil {
		return nil, fmt.Errorf("erro ao marcar processado gatilho %s: %w", iuguFaturaID, err)
	}

	// Salva os dados completos da fatura Iugu em gisp_iugu_faturas_json
	faturaIuguJSON, _ := json.Marshal(faturaIugu)
	if err := repos.Gatilho.SalvarFaturaJSON(tx, iuguFaturaID, faturaIugu, string(faturaIuguJSON)); err != nil {
		logger.Aviso(tagBaixa, "Instancia %d: erro ao salvar JSON fatura %s: %v", instancia.ID, iuguFaturaID, err)
		// Erro não fatal — continua o processamento
	}

	// Lança o valor no caixa
	lancarCaixa(tx, repos, fatura, valorPago, dataHora)

	// Gera o protocolo de baixa e obtém o contrato
	contrato := criarProtocoloBaixa(tx, repos, fatura, valorPago, dataHora, protocolo)

	// Executa o desbloqueio do contrato se aplicável
	resultado := DesbloquearContrato(tx, db, instancia, fatura.ContratoID, dataHora, contrato, repos)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("erro ao commitar transacao: %w", err)
	}

	logger.Sucesso(tagBaixa, "Instancia %d: fatura %d baixada (contrato=%d cliente=%s protocolo=%s)",
		instancia.ID, fatura.ID, fatura.ContratoID, payerName, protocolo)
	return resultado, nil
}

// validarStatusIugu verifica se o status da fatura na Iugu é compatível
// com o status esperado. Para externally_paid, aceita apenas esse status.
// Para os demais, aceita "paid" ou "partially_paid".
func validarStatusIugu(fatura *iugu.FaturaIugu, statusEsperado string) error {
	if statusEsperado == "externally_paid" && fatura.Status != "externally_paid" {
		return fmt.Errorf("status inesperado para externally_paid: %s", fatura.Status)
	}
	if statusEsperado != "externally_paid" && fatura.Status != "paid" && fatura.Status != "partially_paid" {
		return fmt.Errorf("status inesperado: %s", fatura.Status)
	}
	return nil
}

// lancarCaixa registra o lançamento do valor recebido no caixa,
// atualizando o saldo e inserindo o registro no fluxo de caixa.
func lancarCaixa(tx *sql.Tx, repos *Repositorios, fatura entity.Fatura, valorPago, dataHora string) {
	saldoAtual, err := repos.Fatura.BuscarSaldoCaixa(tx, 1)
	if err != nil {
		logger.Aviso(tagBaixa, "Caixa 1 nao encontrado para fatura %d: %v", fatura.ID, err)
		return
	}

	valorNumerico := 0
	_, _ = fmt.Sscanf(helpers.LimparNumero(valorPago), "%d", &valorNumerico)
	novoSaldo := saldoAtual + valorNumerico

	if err := repos.Fatura.AtualizarSaldoCaixa(tx, 1, novoSaldo); err != nil {
		logger.Aviso(tagBaixa, "Erro ao atualizar saldo caixa fatura %d: %v", fatura.ID, err)
		return
	}

	saldoGlobal, err := repos.Fatura.SomarSaldosCaixas(tx)
	if err != nil {
		logger.Aviso(tagBaixa, "Erro ao somar saldos caixas fatura %d: %v", fatura.ID, err)
		return
	}

	descricao := fmt.Sprintf("RECEBIMENTO FAT N %d (B)", fatura.ID)
	token := fmt.Sprintf("%d", rand.Int63())
	seqProtocolo := fmt.Sprintf("%d", helpers.GerarProtocolo(100000, 999999))

	if err := repos.Fatura.InserirFluxoCaixa(tx, saldoGlobal, saldoAtual, novoSaldo, valorNumerico,
		dataHora, token, seqProtocolo, descricao); err != nil {
		logger.Aviso(tagBaixa, "Erro ao inserir fluxo caixa fatura %d: %v", fatura.ID, err)
		return
	}

	logger.Info(tagBaixa, "Caixa fatura %d: saldo %d -> %d (global=%d)", fatura.ID, saldoAtual, novoSaldo, saldoGlobal)
}

// criarProtocoloBaixa gera um registro de protocolo para a baixa da fatura
// em sgp_clientes_contratos_protocolos. Retorna o contrato associado para
// uso no desbloqueio.
func criarProtocoloBaixa(tx *sql.Tx, repos *Repositorios, fatura entity.Fatura, valorPago, dataHora, protocolo string) *entity.Contrato {
	contrato, err := repos.Contrato.BuscarContratoPorID(tx, fatura.ContratoID)
	if err != nil {
		logger.Aviso(tagBaixa, "Contrato %d nao encontrado protocolo fatura %d: %v", fatura.ContratoID, fatura.ID, err)
		return nil
	}

	agoraShort := fuso.Agora().Format("02/01/2006 15:04")
	descricao := fmt.Sprintf("Fatura n %d valor R$ %s valor recebido R$ %s Contrato n %d (%s) baixada em %s",
		fatura.ID, helpers.FormatarMoeda(fatura.Valor), helpers.FormatarMoeda(valorPago),
		contrato.ID, contrato.ClienteNome, agoraShort)

	dadosAntigos, _ := json.Marshal(map[string]interface{}{
		"fatura": map[string]interface{}{"id": fatura.ID, "status": fatura.Status},
	})
	dadosNovos, _ := json.Marshal(map[string]interface{}{
		"fatura": map[string]interface{}{"id": fatura.ID, "status": "Pago"},
	})

	token := fmt.Sprintf("tok_%d", rand.Int63())

	if err := repos.Contrato.InserirProtocolo(tx, token, contrato.Token, contrato.ID,
		protocolo, dataHora, descricao, "Baixa de fatura",
		string(dadosAntigos), string(dadosNovos)); err != nil {
		logger.Aviso(tagBaixa, "Erro ao inserir protocolo baixa fatura %d: %v", fatura.ID, err)
		return contrato
	}

	logger.Info(tagBaixa, "Protocolo baixa %s gerado: fatura=%d contrato=%d cliente=%s",
		protocolo, fatura.ID, contrato.ID, contrato.ClienteNome)
	return contrato
}
