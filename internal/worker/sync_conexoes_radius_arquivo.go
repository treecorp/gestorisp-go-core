package worker

import (
	"database/sql"
	"fmt"
	"strings"

	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
)

var colunasIPv6 = []string{
	"framedipv6pool",
	"framedipv6prefix",
	"delegatedipv6prefix",
	"mikrotikrealm",
}

var colunasObrigatoriasRadacct = []string{
	"radacctid", "acctsessionid", "acctuniqueid", "username",
	"realm", "nasipaddress", "nasportid", "nasporttype",
	"acctstarttime", "acctupdatetime", "acctstoptime",
	"acctinterval", "acctsessiontime", "acctauthentic",
	"connectinfo_start", "connectinfo_stop",
	"acctinputoctets", "acctoutputoctets",
	"calledstationid", "callingstationid",
	"acctterminatecause", "servicetype", "framedprotocol",
	"framedipaddress", "groupname", "contrato_id", "contrato_pop_id",
}

type radacctRecord struct {
	RadAcctID           int64
	AcctSessionID       string
	AcctUniqueID        string
	Username            string
	Realm               sql.NullString
	NASIPAddress        string
	NASPortID           sql.NullString
	NASPortType         sql.NullString
	AcctStartTime       sql.NullTime
	AcctUpdateTime      sql.NullTime
	AcctStopTime        sql.NullTime
	AcctInterval        sql.NullInt64
	AcctSessionTime     sql.NullInt64
	AcctAuthentic       string
	ConnectInfoStart    sql.NullString
	ConnectInfoStop     sql.NullString
	AcctInputOctets     sql.NullInt64
	AcctOutputOctets    sql.NullInt64
	CalledStationID     string
	CallingStationID    string
	AcctTerminateCause  sql.NullString
	ServiceType         sql.NullString
	FramedProtocol      sql.NullString
	FramedIPAddress     sql.NullString
	GroupName           string
	ContratoID          sql.NullInt64
	ContratoPopID       sql.NullInt64
	FramedIPv6Pool      sql.NullString
	FramedIPv6Prefix    sql.NullString
	DelegatedIPv6Prefix sql.NullString
	MikrotikRealm       sql.NullString
}

func HandlerSyncConexoesRadiusArquivo(instancia dominio.Instancia) error {
	tag := "sync_conexoes_radius_arquivo"
	logger.Inicio(tag, "Instancia %d: processando...", instancia.ID)

	db, err := banco.ConectarInstancia(
		instancia.EnvDBHost, instancia.EnvDBPort,
		instancia.EnvDBUser, instancia.EnvDBPass, instancia.EnvDBName,
	)
	if err != nil {
		return fmt.Errorf("falha ao conectar na instancia %d: %w", instancia.ID, err)
	}
	defer banco.FecharConexaoInstancia(db, tag)

	colunasDisponiveis, err := detectarColunasArquivo(db)
	if err != nil {
		return fmt.Errorf("erro ao detectar colunas de radacct_arquivo: %w", err)
	}

	colunasRadacct, err := detectarColunasRadacct(db)
	if err != nil {
		return fmt.Errorf("erro ao detectar colunas de radacct: %w", err)
	}

	registros, err := buscarRadacctPendenteArquivo(db, colunasRadacct)
	if err != nil {
		return fmt.Errorf("erro ao buscar registros pendentes: %w", err)
	}

	if registros == nil {
		logger.Info(tag, "Instancia %d: nenhum registro pendente", instancia.ID)
		return nil
	}

	logger.Info(tag, "Instancia %d: %d registros pendentes", instancia.ID, len(registros))

	migrados := 0
	deletados := 0

	for _, rec := range registros {
		if err := processarRegistro(tag, db, rec, colunasDisponiveis); err != nil {
			logger.Erro(tag, "Instancia %d, radacctid %d: %v", instancia.ID, rec.RadAcctID, err)
			continue
		}
		migrados++
		deletados++
	}

	logger.Sucesso(tag, "Instancia %d: %d migrados, %d deletados", instancia.ID, migrados, deletados)
	return nil
}

func detectarColunasArquivo(db *sql.DB) (map[string]bool, error) {
	query := `SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = (SELECT DATABASE())
		AND TABLE_NAME = 'radacct_arquivo'
		ORDER BY ORDINAL_POSITION`

	linhas, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar INFORMATION_SCHEMA: %w", err)
	}
	defer linhas.Close()

	existentes := make(map[string]bool)
	for linhas.Next() {
		var nome string
		if err := linhas.Scan(&nome); err != nil {
			return nil, fmt.Errorf("erro ao escanear nome da coluna: %w", err)
		}
		existentes[nome] = true
	}
	if err := linhas.Err(); err != nil {
		return nil, err
	}

	for _, col := range colunasIPv6 {
		if !existentes[col] {
			logger.Aviso("sync_conexoes_radius_arquivo", "coluna '%s' ausente em radacct_arquivo - ignorada", col)
		}
	}

	return existentes, nil
}

func detectarColunasRadacct(db *sql.DB) (map[string]bool, error) {
	query := `SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = (SELECT DATABASE())
		AND TABLE_NAME = 'radacct'
		ORDER BY ORDINAL_POSITION`

	linhas, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar INFORMATION_SCHEMA para radacct: %w", err)
	}
	defer linhas.Close()

	existentes := make(map[string]bool)
	for linhas.Next() {
		var nome string
		if err := linhas.Scan(&nome); err != nil {
			return nil, fmt.Errorf("erro ao escanear nome da coluna: %w", err)
		}
		existentes[nome] = true
	}
	if err := linhas.Err(); err != nil {
		return nil, err
	}

	for _, col := range colunasIPv6 {
		if !existentes[col] {
			logger.Aviso("sync_conexoes_radius_arquivo", "coluna IPv6 '%s' ausente em radacct - ignorada no SELECT", col)
		}
	}

	return existentes, nil
}

func buscarRadacctPendenteArquivo(db *sql.DB, colunasRadacct map[string]bool) ([]radacctRecord, error) {
	colunasSELECT := montarListaColunasSELECT(colunasRadacct)

	query := fmt.Sprintf(`SELECT %s FROM radacct
	WHERE acctauthentic = 'RADIUS'
		AND acctstoptime IS NOT NULL
		AND contrato_id IS NOT NULL
		AND (SELECT COUNT(*) FROM radacct
			WHERE acctstoptime IS NOT NULL
			AND contrato_id IS NOT NULL) > 1
	ORDER BY radacctid DESC
	LIMIT 4999`, colunasSELECT)

	linhas, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro na query radacct pendente: %w", err)
	}
	defer linhas.Close()

	var resultado []radacctRecord
	for linhas.Next() {
		var r radacctRecord
		targets := montarScanTargets(&r, colunasRadacct)
		if err := linhas.Scan(targets...); err != nil {
			return nil, fmt.Errorf("erro ao escanear radacct: %w", err)
		}
		resultado = append(resultado, r)
	}

	if len(resultado) == 0 {
		return nil, nil
	}

	return resultado, linhas.Err()
}

func montarListaColunasSELECT(colunasRadacct map[string]bool) string {
	var cols []string
	for _, nome := range colunasObrigatoriasRadacct {
		if colunasRadacct[nome] {
			cols = append(cols, nome)
		}
	}
	for _, nome := range colunasIPv6 {
		if colunasRadacct[nome] {
			cols = append(cols, nome)
		}
	}
	return strings.Join(cols, ", ")
}

func montarScanTargets(r *radacctRecord, colunasRadacct map[string]bool) []interface{} {
	m := map[string]interface{}{
		"radacctid":          &r.RadAcctID,
		"acctsessionid":      &r.AcctSessionID,
		"acctuniqueid":       &r.AcctUniqueID,
		"username":           &r.Username,
		"realm":              &r.Realm,
		"nasipaddress":       &r.NASIPAddress,
		"nasportid":          &r.NASPortID,
		"nasporttype":        &r.NASPortType,
		"acctstarttime":      &r.AcctStartTime,
		"acctupdatetime":     &r.AcctUpdateTime,
		"acctstoptime":       &r.AcctStopTime,
		"acctinterval":       &r.AcctInterval,
		"acctsessiontime":    &r.AcctSessionTime,
		"acctauthentic":      &r.AcctAuthentic,
		"connectinfo_start":  &r.ConnectInfoStart,
		"connectinfo_stop":   &r.ConnectInfoStop,
		"acctinputoctets":    &r.AcctInputOctets,
		"acctoutputoctets":   &r.AcctOutputOctets,
		"calledstationid":    &r.CalledStationID,
		"callingstationid":   &r.CallingStationID,
		"acctterminatecause": &r.AcctTerminateCause,
		"servicetype":        &r.ServiceType,
		"framedprotocol":     &r.FramedProtocol,
		"framedipaddress":    &r.FramedIPAddress,
		"groupname":          &r.GroupName,
		"contrato_id":        &r.ContratoID,
		"contrato_pop_id":    &r.ContratoPopID,
		"framedipv6pool":     &r.FramedIPv6Pool,
		"framedipv6prefix":   &r.FramedIPv6Prefix,
		"delegatedipv6prefix": &r.DelegatedIPv6Prefix,
		"mikrotikrealm":      &r.MikrotikRealm,
	}

	var targets []interface{}
	for _, nome := range colunasObrigatoriasRadacct {
		if colunasRadacct[nome] {
			targets = append(targets, m[nome])
		}
	}
	for _, nome := range colunasIPv6 {
		if colunasRadacct[nome] {
			targets = append(targets, m[nome])
		}
	}
	return targets
}

func processarRegistro(tag string, db *sql.DB, rec radacctRecord, colunasDisponiveis map[string]bool) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("erro ao iniciar transacao: %w", err)
	}
	defer tx.Rollback()

	cols, _ := montarColunasValores(rec, colunasDisponiveis)

	err = inserirRadacctArquivo(tx, cols)
	if err != nil {
		if strings.Contains(err.Error(), "1062") {
			err = atualizarRadacctArquivoPorAcctUniqueID(tx, cols, rec.AcctUniqueID)
		}
		if err != nil {
			return fmt.Errorf("erro ao escrever em radacct_arquivo: %w", err)
		}
	}

	if err := deletarRadacct(tx, rec.RadAcctID); err != nil {
		return fmt.Errorf("erro ao deletar de radacct: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erro ao commitar transacao: %w", err)
	}

	return nil
}

type parColunaValor struct {
	Coluna string
	Valor  interface{}
}

func montarColunasValores(rec radacctRecord, colunasDisponiveis map[string]bool) ([]parColunaValor, string) {
	todos := []parColunaValor{
		{"radacctid", rec.RadAcctID},
		{"acctsessionid", rec.AcctSessionID},
		{"acctuniqueid", rec.AcctUniqueID},
		{"username", rec.Username},
		{"realm", rec.Realm},
		{"nasipaddress", rec.NASIPAddress},
		{"nasportid", rec.NASPortID},
		{"nasporttype", rec.NASPortType},
		{"acctstarttime", rec.AcctStartTime},
		{"acctupdatetime", rec.AcctUpdateTime},
		{"acctstoptime", rec.AcctStopTime},
		{"acctinterval", rec.AcctInterval},
		{"acctsessiontime", rec.AcctSessionTime},
		{"acctauthentic", rec.AcctAuthentic},
		{"connectinfo_start", rec.ConnectInfoStart},
		{"connectinfo_stop", rec.ConnectInfoStop},
		{"acctinputoctets", rec.AcctInputOctets},
		{"acctoutputoctets", rec.AcctOutputOctets},
		{"calledstationid", rec.CalledStationID},
		{"callingstationid", rec.CallingStationID},
		{"acctterminatecause", rec.AcctTerminateCause},
		{"servicetype", rec.ServiceType},
		{"framedprotocol", rec.FramedProtocol},
		{"framedipaddress", rec.FramedIPAddress},
		{"groupname", rec.GroupName},
		{"contrato_id", rec.ContratoID},
		{"contrato_pop_id", rec.ContratoPopID},
		{"framedipv6pool", rec.FramedIPv6Pool},
		{"framedipv6prefix", rec.FramedIPv6Prefix},
		{"delegatedipv6prefix", rec.DelegatedIPv6Prefix},
		{"mikrotikrealm", rec.MikrotikRealm},
	}

	var filtrado []parColunaValor
	for _, p := range todos {
		if colunasDisponiveis[p.Coluna] {
			filtrado = append(filtrado, p)
		}
	}

	var colunas []string
	var valores []interface{}
	for _, p := range filtrado {
		colunas = append(colunas, p.Coluna)
		valores = append(valores, p.Valor)
	}

	return filtrado, strings.Join(colunas, ", ")
}

func inserirRadacctArquivo(tx *sql.Tx, cols []parColunaValor) error {
	var nomes []string
	var placeholders []string
	var valores []interface{}

	for _, p := range cols {
		nomes = append(nomes, p.Coluna)
		placeholders = append(placeholders, "?")
		valores = append(valores, p.Valor)
	}

	query := fmt.Sprintf("INSERT INTO radacct_arquivo (%s) VALUES (%s)",
		strings.Join(nomes, ", "),
		strings.Join(placeholders, ", "))

	_, err := tx.Exec(query, valores...)
	return err
}

func atualizarRadacctArquivo(tx *sql.Tx, cols []parColunaValor, radacctid int64) error {
	var sets []string
	var valores []interface{}

	for _, p := range cols {
		if p.Coluna == "radacctid" {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = ?", p.Coluna))
		valores = append(valores, p.Valor)
	}

	valores = append(valores, radacctid)

	query := fmt.Sprintf("UPDATE radacct_arquivo SET %s WHERE radacctid = ?",
		strings.Join(sets, ", "))

	_, err := tx.Exec(query, valores...)
	return err
}

func atualizarRadacctArquivoPorAcctUniqueID(tx *sql.Tx, cols []parColunaValor, acctUniqueID string) error {
	var sets []string
	var valores []interface{}

	for _, p := range cols {
		if p.Coluna == "acctuniqueid" || p.Coluna == "radacctid" {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = ?", p.Coluna))
		valores = append(valores, p.Valor)
	}

	valores = append(valores, acctUniqueID)

	query := fmt.Sprintf("UPDATE radacct_arquivo SET %s WHERE acctuniqueid = ?",
		strings.Join(sets, ", "))

	_, err := tx.Exec(query, valores...)
	return err
}

func deletarRadacct(tx *sql.Tx, radacctid int64) error {
	query := "DELETE FROM radacct WHERE radacctid = ?"
	_, err := tx.Exec(query, radacctid)
	return err
}
