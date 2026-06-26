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
)

// DesbloquearContrato executa a regra de negócio de desbloqueio de contrato
// após pagamento confirmado. Se o contrato estiver bloqueado, altera seu
// status para "Ativo", remove o bloqueio no radreply, gera protocolo de
// desbloqueio e retorna as informações do POP para desconexão do cliente.
//
// O parâmetro contrato pode ser nil; neste caso o contrato é buscado
// novamente do repositório.
func DesbloquearContrato(
	tx *sql.Tx,
	db *sql.DB,
	instancia entity.Instancia,
	contratoID int,
	dataHora string,
	contrato *entity.Contrato,
	repos *Repositorios,
) *ResultadoBaixa {
	if contrato == nil {
		var err error
		contrato, err = repos.Contrato.BuscarContratoPorID(tx, contratoID)
		if err != nil {
			logger.Aviso(tagBaixa, "Instancia %d: contrato %d nao encontrado desbloqueio: %v", instancia.ID, contratoID, err)
			return nil
		}
	}

	if contrato.Status != "Bloqueado" {
		logger.Info(tagBaixa, "Instancia %d: contrato %d (%s) nao esta bloqueado (status=%s)",
			instancia.ID, contratoID, contrato.ClienteNome, contrato.Status)
		return nil
	}

	logger.Info(tagBaixa, "Instancia %d: desbloqueando contrato %d (%s pppoe=%s)",
		instancia.ID, contratoID, contrato.ClienteNome, contrato.PPPoEUser)

	// Atualiza o status do contrato para Ativo
	if err := repos.Contrato.DesbloquearContrato(tx, contratoID); err != nil {
		logger.Aviso(tagBaixa, "Instancia %d: erro ao desbloquear contrato %d: %v", instancia.ID, contratoID, err)
		return nil
	}

	// Remove o bloqueio no radreply (pgcorte)
	if err := repos.Contrato.RemoverRadReplyCorte(tx, contratoID); err != nil {
		logger.Aviso(tagBaixa, "Instancia %d: erro ao remover radreply corte contrato %d: %v", instancia.ID, contratoID, err)
		// Erro não fatal — continua
	}

	// Gera protocolo de desbloqueio
	agoraShort := fuso.Agora().Format("02/01/2006 15:04")
	descricao := fmt.Sprintf("Contrato n %d (%s) desbloqueado em %s", contratoID, contrato.ClienteNome, agoraShort)
	dadosAntigos, _ := json.Marshal(map[string]interface{}{
		"contrato": map[string]interface{}{"id": contratoID, "status": "Bloqueado"},
	})
	dadosNovos, _ := json.Marshal(map[string]interface{}{
		"contrato": map[string]interface{}{"id": contratoID, "status": "Ativo"},
	})
	token := fmt.Sprintf("tok_%d", rand.Int63())
	bloqProtocolo := fmt.Sprintf("%d", helpers.GerarProtocolo(400000, 499999))

	if err := repos.Contrato.InserirProtocolo(tx, token, contrato.Token, contrato.ID,
		bloqProtocolo, dataHora, descricao, "Desbloqueio de contrato",
		string(dadosAntigos), string(dadosNovos)); err != nil {
		logger.Aviso(tagBaixa, "Instancia %d: erro ao inserir protocolo desbloqueio contrato %d: %v",
			instancia.ID, contratoID, err)
		// Erro não fatal — continua
	}

	// Busca POPs operacionais para obter dados de conexão
	pops, err := repos.Pop.BuscarPopsOperacionais(db)
	if err != nil {
		logger.Aviso(tagBaixa, "Instancia %d: erro ao buscar POPs: %v", instancia.ID, err)
		return nil
	}

	mapaPops := make(map[int]entity.Pop)
	for _, p := range pops {
		mapaPops[p.ID] = p
	}

	pop, ok := mapaPops[contrato.PopID]
	if !ok {
		logger.Aviso(tagBaixa, "Instancia %d: POP %d nao encontrado contrato %d (%s)",
			instancia.ID, contrato.PopID, contratoID, contrato.ClienteNome)
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
