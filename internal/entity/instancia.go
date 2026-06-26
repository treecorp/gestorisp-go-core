package entity

// Instancia representa uma instância GISP com suas credenciais de banco.
type Instancia struct {
	ID        int    `json:"id"`
	Token     string `json:"token"`
	EnvDBHost string `json:"env_dbhost"`
	EnvDBPort string `json:"env_dbport"`
	EnvDBUser string `json:"env_dbuser"`
	EnvDBPass string `json:"env_dbpass"`
	EnvDBName string `json:"env_dbname"`
}

// GetID retorna o identificador numérico da instância.
func (i Instancia) GetID() int {
	return i.ID
}
