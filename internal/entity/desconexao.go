package entity

import "time"

// MensagemDesconexaoContrato representa a mensagem publicada na fila RabbitMQ
// solicitando a desconexão de um contrato do POP.
type MensagemDesconexaoContrato struct {
	Instancia   Instancia `json:"instancia"`
	ContratoID  int       `json:"contrato_id"`
	ClienteNome string    `json:"cliente_nome"`
	PPPoEUser   string    `json:"pppoe_user"`
	PopIPv4     string    `json:"pop_ipv4"`
	PopPort     string    `json:"pop_port"`
	PopUser     string    `json:"pop_user"`
	PopPass     string    `json:"pop_pass"`
	CriadoEm    string    `json:"criado_em"`
}

// Expirada retorna true se a mensagem foi criada há mais de 24 horas.
func (m MensagemDesconexaoContrato) Expirada() bool {
	if m.CriadoEm == "" {
		return false
	}
	criado, err := time.Parse(time.RFC3339, m.CriadoEm)
	if err != nil {
		return false
	}
	return time.Since(criado) > 24*time.Hour
}
