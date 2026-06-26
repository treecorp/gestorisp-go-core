package bloqueio

import (
	"database/sql"

	"gestor/internal/entity"
	"gestor/internal/repositorio"
)

// Queryer é a interface para operações de consulta (QueryRow), satisfeita
// por *sql.DB e *sql.Tx. Reexportada de internal/repositorio para
// conveniência do pacote.
type Queryer = repositorio.Queryer

// ContratoRepo define as operações de persistência relacionadas a contratos
// necessárias para o serviço de bloqueio.
type ContratoRepo interface {
	// BuscarContratoPorID busca um contrato pelo ID com LEFT JOIN
	// sgp_clientes_new para obter o nome do cliente.
	BuscarContratoPorID(q Queryer, contratoID int) (*entity.Contrato, error)
}

// BloqueioRepo define as operações de persistência relacionadas a bloqueio
// de clientes (faturas vencidas, desbloqueio por confiança, etc.).
type BloqueioRepo interface {
	// BuscarFaturasVencidas retorna faturas vencidas não pagas com base
	// nos critérios do cron (isento = 'Não', status = 'Pendente',
	// vencimento entre 2018-06-01 e a data atual).
	BuscarFaturasVencidas(db *sql.DB, diasBloqueio int) ([]entity.Fatura, error)

	// LerDiasBloqueio lê o valor de dias_bloqueio da tabela
	// sgp_parametros. Retorna 5 como valor padrão caso a consulta falhe.
	LerDiasBloqueio(db *sql.DB) int

	// LerDesbloqueioConfianca verifica se o contrato possui um
	// desbloqueio por confiança ativo. Retorna os dados do desbloqueio
	// ou nil se não houver registro vigente.
	LerDesbloqueioConfianca(db *sql.DB, contratoID int) (*repositorio.DesbloqueioConfianca, error)
}

// Repositorios agrupa as interfaces de repositório necessárias para o
// service de bloqueio, facilitando injeção de dependência e testes.
type Repositorios struct {
	Contrato ContratoRepo
	Bloqueio BloqueioRepo
}

// ClienteBloqueado representa o resultado de um bloqueio aplicado a um
// contrato. Contém os dados necessários para publicar a mensagem de
// desconexão na fila RabbitMQ.
type ClienteBloqueado struct {
	ContratoID  int
	PPPoEUser   string
	PopIPv4     string
	PopPort     string
	PopUser     string
	PopPass     string
	ClienteNome string
}
