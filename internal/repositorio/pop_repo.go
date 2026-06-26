package repositorio

import (
	"database/sql"
	"fmt"

	"gestor/internal/entity"
)

// BuscarPopsOperacionais retorna todos os POPs com status "OPERACIONAL".
// Extraido de carregarPops (cron_1.go) + BuscarPopsOperacionais (gispinstancia.go).
func BuscarPopsOperacionais(db *sql.DB) ([]entity.Pop, error) {
	query := `SELECT id, ipv4, api_port, user, pass, status, status_timeout, status_timeout_data_hora
		FROM sgp_pop WHERE status = 'OPERACIONAL' ORDER BY id ASC`

	linhas, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("buscar pops operacionais: %w", err)
	}
	defer linhas.Close()

	var pops []entity.Pop
	for linhas.Next() {
		var p entity.Pop
		if err := linhas.Scan(
			&p.ID, &p.IPv4, &p.APIPort, &p.User, &p.Pass,
			&p.Status, &p.StatusTimeout, &p.StatusTimeoutDataHora,
		); err != nil {
			return nil, fmt.Errorf("buscar pops operacionais: erro ao escanear pop: %w", err)
		}
		pops = append(pops, p)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("buscar pops operacionais: erro na iteracao: %w", err)
	}

	return pops, nil
}

// AtualizarStatusTimeout marca o POP com o timeout informado, atualizando
// status_timeout e opcionalmente status_timeout_data_hora.
// Extraido de AtualizarStatusTimeout (gispinstancia.go).
func AtualizarStatusTimeout(db *sql.DB, popID int, timeout int, dataHora *string) error {
	if dataHora != nil && *dataHora != "" {
		query := `UPDATE sgp_pop SET status_timeout = ?, status_timeout_data_hora = ? WHERE id = ?`
		_, err := db.Exec(query, timeout, *dataHora, popID)
		if err != nil {
			return fmt.Errorf("atualizar status timeout pop %d: %w", popID, err)
		}
	} else {
		query := `UPDATE sgp_pop SET status_timeout = ?, status_timeout_data_hora = NULL WHERE id = ?`
		_, err := db.Exec(query, timeout, popID)
		if err != nil {
			return fmt.Errorf("atualizar status timeout pop %d: %w", popID, err)
		}
	}

	return nil
}
