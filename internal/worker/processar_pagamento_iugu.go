package worker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
	"gestor/internal/pagamento"
)

const tagPagamento = "processar_pagamento_iugu"

func HandlerProcessarPagamentoIugu(body []byte, rabbit *mensageria.RabbitMQ) error {
	jsonBytes, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		return fmt.Errorf("erro ao decodificar base64: %w", err)
	}

	var msg dominio.MensagemPagamentoIugu
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

	resultado, err := pagamento.ProcessarPagamento(db, instancia, msg.Data, iuguID, status)
	if err != nil {
		logger.Erro(tagPagamento, "Instancia %d: erro ao processar pagamento %s: %v", instancia.ID, iuguID, err)
		return err
	}

	if rabbit != nil && resultado != nil && resultado.PrecisaDesconectar {
		desconexao := dominio.MensagemDesconexaoContrato{
			Instancia:   instancia,
			ContratoID:  resultado.ContratoID,
			ClienteNome: resultado.ClienteNome,
			PPPoEUser:   resultado.PPPoEUser,
			PopIPv4:     resultado.PopIPv4,
			PopPort:     resultado.PopPort,
			PopUser:     resultado.PopUser,
			PopPass:     resultado.PopPass,
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
