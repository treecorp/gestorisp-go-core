package dominio

import "database/sql"

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
