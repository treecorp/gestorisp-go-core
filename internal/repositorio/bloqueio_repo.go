package repositorio

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gestor/internal/entity"
)

// DesbloqueioConfianca representa a configuracao de desbloqueio
// automatico por confianca para um contrato. Ativo indica se existe
// um registro de confianca vigente e Dias indica quantos dias restam
// ate a expiracao do periodo de confianca (valores negativos indicam
// periodo expirado).
type DesbloqueioConfianca struct {
	Ativo bool
	Dias  int
}

// BuscarFaturasVencidas retorna faturas vencidas nao pagas com base
// nos criterios do cron (isento = 'Nao', status = 'Pendente',
// vencimento entre 2018-06-01 e a data atual).
// O parametro diasBloqueio e reservado para uso futuro do caller.
// Extraído de listar_clientes_vencidos.go: buscarFaturasVencidas.
func BuscarFaturasVencidas(db *sql.DB, diasBloqueio int) ([]entity.Fatura, error) {
	dataFinal := time.Now().Format("2006-01-02")
	rows, err := db.Query(`
		SELECT f.id, f.contrato_id, f.contrato_token, f.cliente_token,
		       DATE_FORMAT(f.vencimento, '%Y-%m-%d') AS vencimento
		FROM sgp_clientes_faturas f
		INNER JOIN sgp_clientes_contratos c ON c.id = f.contrato_id
		WHERE c.isento = 'Não'
		  AND f.status = 'Pendente'
		  AND f.vencimento BETWEEN '2018-06-01' AND ?
		  AND f.vencimento < ?
		ORDER BY f.vencimento ASC
	`, dataFinal, dataFinal)
	if err != nil {
		return nil, fmt.Errorf("buscar faturas vencidas: %w", err)
	}
	defer rows.Close()

	var faturas []entity.Fatura
	for rows.Next() {
		var f entity.Fatura
		if err := rows.Scan(&f.ID, &f.ContratoID, &f.ContratoToken,
			&f.ClienteToken, &f.DataVencimento); err != nil {
			return nil, fmt.Errorf("escanear fatura vencida: %w", err)
		}
		faturas = append(faturas, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar faturas vencidas: %w", err)
	}
	if faturas == nil {
		return nil, nil
	}
	return faturas, nil
}

// LerDesbloqueioConfianca verifica se o contrato possui um
// desbloqueio por confianca ativo. Retorna os dados do desbloqueio
// ou nil se nao houver registro vigente.
// Extraído de listar_clientes_vencidos.go: lerDesbloqueioConfianca.
func LerDesbloqueioConfianca(db *sql.DB, contratoID int) (*DesbloqueioConfianca, error) {
	var id int
	var dataHoraBloqueio string
	err := db.QueryRow(`
		SELECT id,
		       DATE_FORMAT(data_hora_bloqueio, '%Y-%m-%d %H:%i:%s') AS data_hora_bloqueio
		FROM sgp_clientes_contratos_desbloqueio_confianca
		WHERE contrato_id = ? AND status = 'Ativo'
		ORDER BY id DESC LIMIT 1
	`, contratoID).Scan(&id, &dataHoraBloqueio)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ler desbloqueio confianca: %w", err)
	}

	dataBloqueio, err := time.Parse("2006-01-02 15:04:05", dataHoraBloqueio)
	if err != nil {
		// fallback: tenta apenas a data
		dataBloqueio, err = time.Parse("2006-01-02", dataHoraBloqueio[:10])
		if err != nil {
			// nao foi possivel interpretar a data; retorna como ativo com dias zero
			return &DesbloqueioConfianca{Ativo: true, Dias: 0}, nil
		}
	}

	dias := int(time.Until(dataBloqueio).Hours() / 24)
	return &DesbloqueioConfianca{Ativo: true, Dias: dias}, nil
}

// LerDiasBloqueio le o valor de dias_bloqueio da tabela
// sgp_parametros. Faz parse manual para suportar valores armazenados
// como varchar (HOTFIX-004). Retorna 5 como valor padrao caso a
// consulta falhe ou o valor seja invalido.
// Extraído de listar_clientes_vencidos.go: lerDiasBloqueio.
func LerDiasBloqueio(db *sql.DB) int {
	var diasStr sql.NullString
	err := db.QueryRow("SELECT dias_bloqueio FROM sgp_parametros LIMIT 1").Scan(&diasStr)
	if err != nil || !diasStr.Valid || strings.TrimSpace(diasStr.String) == "" {
		return 5
	}

	// Parse manual para varchar (HOTFIX-004)
	dias, err := strconv.Atoi(strings.TrimSpace(diasStr.String))
	if err != nil {
		return 5
	}
	return dias
}
