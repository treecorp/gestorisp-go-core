package entity

import "database/sql"

// Pop representa um ponto de presença (POP) com suas credenciais de acesso
// e parâmetros de timeout para verificação de status.
type Pop struct {
	ID                   int            `json:"id"`
	IPv4                 string         `json:"ipv4"`
	APIPort              string         `json:"api_port"`
	User                 string         `json:"user"`
	Pass                 string         `json:"pass"`
	Status               string         `json:"status"`
	StatusTimeout        int            `json:"status_timeout"`
	StatusTimeoutDataHora sql.NullString `json:"status_timeout_data_hora"`
}
