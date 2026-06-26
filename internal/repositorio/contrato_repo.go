package repositorio

import (
	"database/sql"
	"fmt"

	"gestor/internal/entity"
)

// Queryer é a interface que define o metodo QueryRow, implementada
// por *sql.DB e *sql.Tx. Permite que as funcoes de consulta aceitem
// ambos os tipos.
type Queryer interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

// BuscarContratoPorID busca um contrato pelo ID com LEFT JOIN
// sgp_clientes_new para obter o nome do cliente. Usa COALESCE para
// garantir cliente_id = 0 quando nao houver cliente vinculado e
// join por cliente_token (HOTFIX-007).
// Extraído de processar.go: buscarContrato.
func BuscarContratoPorID(q Queryer, contratoID int) (*entity.Contrato, error) {
	var c entity.Contrato
	err := q.QueryRow(`SELECT c.id, c.token, c.status,
		COALESCE(c.cliente_id, 0),
		COALESCE(cli.pf_nome, cli.pj_razao_social, 'N/D') AS cliente_nome,
		c.cliente_token, c.pop_id, c.pppoe_user
		FROM sgp_clientes_contratos c
		LEFT JOIN sgp_clientes_new cli ON cli.token = c.cliente_token
		WHERE c.id = ?`, contratoID).Scan(
		&c.ID, &c.Token, &c.Status, &c.ClienteID,
		&c.ClienteNome, &c.ClienteToken, &c.PopID, &c.PPPoEUser,
	)
	if err != nil {
		return nil, fmt.Errorf("buscar contrato por ID: %w", err)
	}
	return &c, nil
}

// DesbloquearContrato atualiza o status do contrato para "Ativo",
// removendo o bloqueio. Executado dentro de uma transacao.
// Extraído de processar.go: desbloquearContratoDB (parte DB).
func DesbloquearContrato(tx *sql.Tx, contratoID int) error {
	_, err := tx.Exec(`UPDATE sgp_clientes_contratos
		SET status = 'Ativo'
		WHERE id = ?`, contratoID)
	if err != nil {
		return fmt.Errorf("desbloquear contrato: %w", err)
	}
	return nil
}
