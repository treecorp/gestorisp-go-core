package entity

import "database/sql"

// Contrato representa um contrato de cliente no sistema GISP.
//
// Mapeia todos os campos da tabela sgp_clientes_contratos e informações
// relacionadas ao POP de conexão.
type Contrato struct {
	ID                 int            `json:"id"`
	Token              string         `json:"token"`
	Status             string         `json:"status"`
	ClienteID          int            `json:"cliente_id"`
	ClienteNome        string         `json:"cliente_nome"`
	ClienteToken       string         `json:"cliente_token"`
	PopID              int            `json:"pop_id"`
	PPPoEUser          string         `json:"pppoe_user"`
	PopIPv4            string         `json:"pop_ipv4"`
	PopPort            string         `json:"pop_port"`
	PopUser            string         `json:"pop_user"`
	PopPass            string         `json:"pop_pass"`
	DataHora           string         `json:"data_hora"`
	DataHoraUltConexao sql.NullString `json:"data_hora_ult_conexao"`
}

// EstaBloqueado retorna true se o contrato estiver com status de bloqueio.
func (c *Contrato) EstaBloqueado() bool {
	return c.Status == "Bloqueado"
}

// Desbloquear altera o status do contrato para "Ativo" apenas em memória.
func (c *Contrato) Desbloquear() {
	c.Status = "Ativo"
}
