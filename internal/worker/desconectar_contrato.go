package worker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"gestor/internal/dominio"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/routeros"
)

const tagDesconexao = "desconectar_contrato"

func HandlerDesconectarContrato(body []byte) error {
	jsonBytes, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		return fmt.Errorf("erro ao decodificar base64: %w", err)
	}

	var msg dominio.MensagemDesconexaoContrato
	if err := json.Unmarshal(jsonBytes, &msg); err != nil {
		return fmt.Errorf("erro ao decodificar JSON: %w", err)
	}

	if msg.Expirada() {
		logger.Aviso(tagDesconexao, "Instancia %d: mensagem de desconexao expirou (>24h). Descartando.", msg.Instancia.ID)
		return nil
	}

	logger.Info(tagDesconexao, "Instancia %d: desconectando contrato %d (%s) pppoe=%s POP=%s:%s",
		msg.Instancia.ID, msg.ContratoID, msg.ClienteNome, msg.PPPoEUser, msg.PopIPv4, msg.PopPort)

	conn, err := routeros.Conectar(routeros.DadosConexao{
		IPv4: msg.PopIPv4,
		Port: msg.PopPort,
		User: msg.PopUser,
		Pass: msg.PopPass,
	})
	if err != nil {
		logger.Aviso(tagDesconexao, "Instancia %d: POP %s:%s inacessivel: %v. Reintentando...",
			msg.Instancia.ID, msg.PopIPv4, msg.PopPort, err)
		return fmt.Errorf("POP %s:%s inacessivel: %w", msg.PopIPv4, msg.PopPort, err)
	}
	defer conn.Close()

	ativo, sessionID, err := routeros.VerificarUsuarioAtivo(conn, msg.PPPoEUser)
	if err != nil {
		logger.Aviso(tagDesconexao, "Instancia %d: erro ao consultar %s no POP %s: %v. Reintentando...",
			msg.Instancia.ID, msg.PPPoEUser, msg.PopIPv4, err)
		return fmt.Errorf("erro ao consultar %s no POP %s: %w", msg.PPPoEUser, msg.PopIPv4, err)
	}

	if !ativo {
		logger.Info(tagDesconexao, "Instancia %d: %s ja desconectado (contrato %d, %s)",
			msg.Instancia.ID, msg.PPPoEUser, msg.ContratoID, msg.ClienteNome)
		return nil
	}

	if err := routeros.DesconectarUsuario(conn, sessionID); err != nil {
		logger.Aviso(tagDesconexao, "Instancia %d: erro ao desconectar %s do POP %s: %v. Reintentando...",
			msg.Instancia.ID, msg.PPPoEUser, msg.PopIPv4, err)
		return fmt.Errorf("erro ao desconectar %s do POP %s: %w", msg.PPPoEUser, msg.PopIPv4, err)
	}

	logger.Sucesso(tagDesconexao, "Instancia %d: %s desconectado do POP %s:%s (contrato %d, %s)",
		msg.Instancia.ID, msg.PPPoEUser, msg.PopIPv4, msg.PopPort, msg.ContratoID, msg.ClienteNome)
	return nil
}
