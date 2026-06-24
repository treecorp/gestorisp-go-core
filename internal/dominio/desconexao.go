package dominio

type MensagemDesconexaoContrato struct {
	Instancia   Instancia `json:"instancia"`
	ContratoID  int       `json:"contrato_id"`
	ClienteNome string    `json:"cliente_nome"`
	PPPoEUser   string    `json:"pppoe_user"`
	PopIPv4     string    `json:"pop_ipv4"`
	PopPort     string    `json:"pop_port"`
	PopUser     string    `json:"pop_user"`
	PopPass     string    `json:"pop_pass"`
}
