package repositorio

import (
	"database/sql"
	"fmt"
	"time"
)

// InserirGatilho registra um gatilho de pagamento Iugu com status
// "Processando". Usa INSERT com ON DUPLICATE KEY para garantir
// idempotência (a tabela gisp_iugu_gatilhos nao possui PK em id).
// Extraído de processar.go (INSERT inicial + marcarProcessando).
func InserirGatilho(tx *sql.Tx, iuguFaturaID string, statusEsperado string) error {
	agora := time.Now().Format("2006-01-02 15:04:05")

	switch statusEsperado {
	case "paid":
		_, err := tx.Exec(`INSERT INTO gisp_iugu_gatilhos
			(id, gisp_exec_status, datetime_received)
			VALUES (?, 'Processando', ?)
			ON DUPLICATE KEY UPDATE id = id`,
			iuguFaturaID, agora)
		if err != nil {
			return fmt.Errorf("inserir gatilho (paid): %w", err)
		}
	case "partially_paid":
		_, err := tx.Exec(`INSERT INTO gisp_iugu_gatilhos
			(id, gisp_exec_status, datetime_received)
			VALUES (?, 'Processando', ?)
			ON DUPLICATE KEY UPDATE id = id`,
			iuguFaturaID, agora)
		if err != nil {
			return fmt.Errorf("inserir gatilho (partially_paid): %w", err)
		}
	default:
		_, err := tx.Exec(`INSERT INTO gisp_iugu_gatilhos
			(id, gisp_exec_status, datetime_received)
			VALUES (?, 'Processando', ?)
			ON DUPLICATE KEY UPDATE id = id`,
			iuguFaturaID, agora)
		if err != nil {
			return fmt.Errorf("inserir gatilho (default): %w", err)
		}
	}

	return nil
}

// MarcarProcessado atualiza o gatilho como processado, definindo
// gisp_exec='1', gisp_exec_status='Processado', o protocolo de retorno
// e a data/hora de processamento. Executado dentro de transacao.
// Extraído de processar.go: marcarProcessado.
func MarcarProcessado(tx *sql.Tx, iuguFaturaID string, status string, protocolo string) error {
	agora := time.Now().Format("2006-01-02 15:04:05")

	switch status {
	case "paid":
		_, err := tx.Exec(`UPDATE gisp_iugu_gatilhos
			SET gisp_exec = '1',
			    gisp_exec_status = 'Processado',
			    gisp_exec_return = ?,
			    datetime_processed = ?
			WHERE id = ?
			  AND status = 'paid'
			  AND event = 'invoice.status_changed'
			  AND gisp_exec = '0'`,
			protocolo, agora, iuguFaturaID)
		if err != nil {
			return fmt.Errorf("marcar processado (paid): %w", err)
		}

	case "partially_paid":
		_, err := tx.Exec(`UPDATE gisp_iugu_gatilhos
			SET gisp_exec = '1',
			    gisp_exec_status = 'Processado',
			    gisp_exec_return = ?,
			    datetime_processed = ?
			WHERE id = ?
			  AND status = 'partially_paid'
			  AND event = 'invoice.status_changed'
			  AND gisp_exec = '0'`,
			protocolo, agora, iuguFaturaID)
		if err != nil {
			return fmt.Errorf("marcar processado (partially_paid): %w", err)
		}

	default:
		_, err := tx.Exec(`UPDATE gisp_iugu_gatilhos
			SET gisp_exec = '1',
			    gisp_exec_status = 'Processado',
			    gisp_exec_return = ?,
			    datetime_processed = ?
			WHERE id = ?
			  AND status = 'externally_paid'
			  AND event = 'invoice.status_changed'
			  AND gisp_exec = '0'`,
			protocolo, agora, iuguFaturaID)
		if err != nil {
			return fmt.Errorf("marcar processado (default): %w", err)
		}
	}

	return nil
}

// MarcarErroGatilho registra erro no processamento do gatilho fora de
// transacao. Define gisp_exec='1', gisp_exec_status com o codigo de
// erro e gisp_exec_return com a mensagem descritiva.
// Extraído de processar.go: marcarErroGatilho.
func MarcarErroGatilho(db *sql.DB, iuguFaturaID string, status string, codErro string, msg string) error {
	agora := time.Now().Format("2006-01-02 15:04:05")

	switch status {
	case "paid":
		_, err := db.Exec(`UPDATE gisp_iugu_gatilhos
			SET gisp_exec = '1',
			    gisp_exec_status = ?,
			    gisp_exec_return = ?,
			    datetime_processed = ?
			WHERE id = ?
			  AND status = 'paid'
			  AND event = 'invoice.status_changed'
			  AND gisp_exec = '0'`,
			codErro, msg, agora, iuguFaturaID)
		if err != nil {
			return fmt.Errorf("marcar erro gatilho (paid): %w", err)
		}

	case "partially_paid":
		_, err := db.Exec(`UPDATE gisp_iugu_gatilhos
			SET gisp_exec = '1',
			    gisp_exec_status = ?,
			    gisp_exec_return = ?,
			    datetime_processed = ?
			WHERE id = ?
			  AND status = 'partially_paid'
			  AND event = 'invoice.status_changed'
			  AND gisp_exec = '0'`,
			codErro, msg, agora, iuguFaturaID)
		if err != nil {
			return fmt.Errorf("marcar erro gatilho (partially_paid): %w", err)
		}

	default:
		_, err := db.Exec(`UPDATE gisp_iugu_gatilhos
			SET gisp_exec = '1',
			    gisp_exec_status = ?,
			    gisp_exec_return = ?,
			    datetime_processed = ?
			WHERE id = ?
			  AND status = 'externally_paid'
			  AND event = 'invoice.status_changed'
			  AND gisp_exec = '0'`,
			codErro, msg, agora, iuguFaturaID)
		if err != nil {
			return fmt.Errorf("marcar erro gatilho (default): %w", err)
		}
	}

	return nil
}
