package config

import "os"

// Config contem todas as configuracoes do sistema (banco, mensageria, etc.)
type Config struct {
	Banco               BancoConfig
	RabbitMQ            RabbitMQConfig
	CI3EncryptionKey    string
	DBInstanciaHostDev  string
	DBInstanciaPortaDev string
	DashboardPort       string
	DashboardIngestURL  string
}

type BancoConfig struct {
	Host     string
	Porta    string
	Usuario  string
	Senha    string
	Database string
}

type RabbitMQConfig struct {
	Host    string
	Porta   string
	Usuario string
	Senha   string
}

// Carregar le as variaveis de ambiente e retorna a configuracao
// TODO: migrar para um provedor de configuracao externo (Vault, AWS Secrets, etc.)
func Carregar() *Config {
	return &Config{
		Banco: BancoConfig{
			Host:     obterEnv("DB_GISPADM_HOST", "177.136.249.51"),
			Porta:    obterEnv("DB_GISPADM_PORT", "31034"),
			Usuario:  obterEnv("DB_GISPADM_USER", "gestorisp"),
			Senha:    obterEnv("DB_GISPADM_PASS", "WM33223200kl**"),
			Database: obterEnv("DB_GISPADM_DBNAME", "gisp_adm"),
		},
		RabbitMQ: RabbitMQConfig{
			Host:    obterEnv("RABBITMQ_HOST", "172.16.12.10"),
			Porta:   obterEnv("RABBITMQ_PORT", "31837"),
			Usuario: obterEnv("RABBITMQ_USER", "guest"),
			Senha:   obterEnv("RABBITMQ_PASS", "guest"),
		},
		CI3EncryptionKey:    obterEnv("CI3_ENCRYPTION_KEY", "sjlkjl32oiPOIjkl2"),
		DBInstanciaHostDev:  obterEnv("DB_INSTANCIA_HOST_DEV", ""),
		DBInstanciaPortaDev: obterEnv("DB_INSTANCIA_PORT_DEV", ""),
		DashboardPort:       obterEnv("DASHBOARD_PORT", "8080"),
		DashboardIngestURL:  obterEnv("DASHBOARD_INGEST_URL", "http://localhost:8080"),
	}
}

func obterEnv(chave, padrao string) string {
	if valor := os.Getenv(chave); valor != "" {
		return valor
	}
	return padrao
}
