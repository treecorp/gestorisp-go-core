package worker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gestor/internal/entity"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
	"gestor/internal/lib/iugu"
	"gestor/internal/service/pagamento"
)

const tagPagamento = "processar_pagamento_iugu"

// HandlerProcessarPagamentoIugu processa uma mensagem de pagamento Iugu
// recebida da fila RabbitMQ. Conecta na instancia, cria os repositorios
// necessarios e delega o processamento ao service de pagamento.
func HandlerProcessarPagamentoIugu(body []byte, rabbit *mensageria.RabbitMQ) error {
	jsonBytes, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		return fmt.Errorf("erro ao decodificar base64: %w", err)
	}

	var msg entity.MensagemPagamentoIugu
	if err := json.Unmarshal(jsonBytes, &msg); err != nil {
		return fmt.Errorf("erro ao decodificar JSON: %w", err)
	}

	instancia := msg.Instancia
	iuguID := msg.Data["id"]
	status := msg.Data["status"]

	logger.Info(tagPagamento, "Processando pagamento: instancia=%d iugu_fatura=%s status=%s tentativa=%d",
		instancia.ID, iuguID, status, msg.Tentativa)

	db, err := banco.ConectarInstancia(
		instancia.EnvDBHost, instancia.EnvDBPort,
		instancia.EnvDBUser, instancia.EnvDBPass, instancia.EnvDBName,
	)
	if err != nil {
		logger.Erro(tagPagamento, "Instancia %d: erro ao conectar banco: %v", instancia.ID, err)
		return fmt.Errorf("erro ao conectar banco instancia %d: %w", instancia.ID, err)
	}
	defer banco.FecharConexaoInstancia(db, tagPagamento)

	// Wire dependencies for service layer
	repos := &pagamento.Repositorios{
		Fatura:   &pagamentoFaturaAdapter{},
		Contrato: &pagamentoContratoAdapter{},
		Gatilho:  &pagamentoGatilhoAdapter{},
		Pop:      &pagamentoPopAdapter{},
	}

	// Create Iugu client
	apiURL := os.Getenv("IUGU_API_URL")
	if apiURL == "" {
		apiURL = "https://api.iugu.com/v1"
	}
	apiKey := os.Getenv("IUGU_API_KEY")
	iuguCli := &pagamentoIuguClientAdapter{
		apiURL: apiURL,
		apiKey: apiKey,
	}

	resultado, err := pagamento.ProcessarPagamento(repos, iuguCli, db, instancia, msg.Data, iuguID, status, msg.Event)
	if err != nil {
		logger.Erro(tagPagamento, "Instancia %d: erro ao processar pagamento %s: %v", instancia.ID, iuguID, err)
		return err
	}

	if rabbit != nil && resultado != nil && resultado.PrecisaDesconectar {
		desconexao := entity.MensagemDesconexaoContrato{
			Instancia:   instancia,
			ContratoID:  resultado.ContratoID,
			ClienteNome: resultado.ClienteNome,
			PPPoEUser:   resultado.PPPoEUser,
			PopIPv4:     resultado.PopIPv4,
			PopPort:     resultado.PopPort,
			PopUser:     resultado.PopUser,
			PopPass:     resultado.PopPass,
			CriadoEm:    time.Now().Format(time.RFC3339),
		}

		logger.Info(tagPagamento, "Instancia %d: publicando desconexao do contrato %d (%s) na fila desconectar_contrato",
			instancia.ID, resultado.ContratoID, resultado.PPPoEUser)

		if err := rabbit.PublicarMensagem("desconectar_contrato", desconexao); err != nil {
			logger.Aviso(tagPagamento, "Instancia %d: erro ao publicar desconexao: %v", instancia.ID, err)
		}
	}

	logger.Sucesso(tagPagamento, "Instancia %d: pagamento %s processado com sucesso", instancia.ID, iuguID)
	return nil
}

// pagamentoIuguClientAdapter implementa pagamento.ClienteIugu usando o cliente Iugu padrao.
type pagamentoIuguClientAdapter struct {
	apiURL string
	apiKey string
}

func (a *pagamentoIuguClientAdapter) ConsultarFatura(faturaID string) (*iugu.FaturaIugu, error) {
	cliente := iugu.NovoClienteIugu(a.apiURL, a.apiKey)
	return iugu.ConsultarFatura(cliente, faturaID)
}

// Ensure interface compliance
var _ pagamento.ClienteIugu = (*pagamentoIuguClientAdapter)(nil)
