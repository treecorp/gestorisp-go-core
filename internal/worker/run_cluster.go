package worker

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
)

const (
	imgOnline = "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAQDAwMDAwQDAwQFBAMEBQcFBAQFBwgGBgcGBggKCAgICAgICggKCgsKCggNDQ4ODQ0SEhISEhQUFBQUFBQUFBT/2wBDAQUFBQgHCA8KCg8SDwwPEhYVFRUVFhYUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBT/wAARCAAKAAoDAREAAhEBAxEB/8QAFwAAAwEAAAAAAAAAAAAAAAAAAwUGCP/EACAQAAMAAwACAgMAAAAAAAAAAAECAwQFBgAREkETITH/xAAZAQACAwEAAAAAAAAAAAAAAAACBAEFBgf/xAAfEQACAgICAwEAAAAAAAAAAAABAgAEAxExYSJCUbH/2gAMAwEAAhEDEQA/AGfUdP00Om3Mp7jYySexyUSa5NQFCWb4qAG9AL9eVjMdziN69YFjIBkceZ9j9Pc0DzBtk83psnIysil7a7GpWjULMzvJSSSfZJJ8bXib2jtq+MknZRfyC3HM83TI/PTT6972ZnrRsaRZ3b9lmPx9kk/fkFRBsUq5bZxrs9CUODKUcLGjFFnGcUSc0AVVVVAAAH8A8IR7EAEAHGp//9k="

	imgOffline = "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAQDAwMDAwQDAwQFBAMEBQcFBAQFBwgGBgcGBggKCAgICAgICggKCgsKCggNDQ4ODQ0SEhISEhQUFBQUFBQUFBT/2wBDAQUFBQgHCA8KCg8SDwwPEhYVFRUVFhYUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBT/wAARCAAKAAoDAREAAhEBAxEB/8QAFwAAAwEAAAAAAAAAAAAAAAAAAwYHCP/EACEQAAICAgIBBQAAAAAAAAAAAAIDAQQFBgARMRIUISJB/8QAGgEAAQUBAAAAAAAAAAAAAAAABgECAwQFB//EACMRAAEDAwMFAQAAAAAAAAAAAAEAAgMEERIhMVEiMkFxkbH/2gAMAwEAAhEDEQA/AJ1u+7brW3TY66dhzCFJzFxa0hdsCICuwfpGIg+ogfzrxwQmmfmdTvyulU1NEYm9Le0eBwtZaTNi7puu3bd6463YxFJz2m0iM2MrgRERT3MzMz5nm1DqweghapsJXAAWyP6g7DpenOt+6dr2JZZsExthx0kEbGH9iIykOymZnuZniPiZwE6GplAtk76U14tCK2Mp166gTXTXUtSljAgACEQIiMfEREeI5M3ZVXm7iSv/2Q=="

	imgBloqueado = "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAQDAwMDAwQDAwQFBAMEBQcFBAQFBwgGBgcGBggKCAgICAgICggKCgsKCggNDQ4ODQ0SEhISEhQUFBQUFBQUFBT/2wBDAQUFBQgHCA8KCg8SDwwPEhYVFRUVFhYUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBT/wAARCAAKAAoDAREAAhEBAxEB/8QAFgABAQEAAAAAAAAAAAAAAAAACAQG/8QAHxAAAgICAwEBAQAAAAAAAAAAAgMBBAUHAAYRQRQx/8QAFAEBAAAAAAAAAAAAAAAAAAAAAP/EABQRAQAAAAAAAAAAAAAAAAAAAAD/2gAMAwEAAhEDEQA/AClsjY+xqexe3Va/bM/VSjP5BSq68jaAFAm2yAARFnkQHyI/nAdmuJtZLXnUsjfyV+xet4DHWLL2vI2Ma2qsjMiL2Skin2ZmeBL2zXWvrF79tjqeBbdtm19qweOqk1rWekRsOV+kRFPszP3gbrC1q1PDY+pUStFRFRKkIUMAsFguBEAEfIiBiPIiOB//2Q=="

	imgDesconexao = "data:image/gif;base64,R0lGODlhCgAKANUAAPP5//v12f/z2f/tsf/ts//trf/tqf/rrf/rn//nvf/rof/pqf/pr//prf/rnf/pp//nqf/nof/no/vpk//nnf/nj/nnmf/lm//lkf/jn//li//jl//jmf/jnf/hmf/jj//hn//hdPvfcP+7AgC5Wv4BAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH/C05FVFNDQVBFMi4wAwEAAAAh+QQFZAAlACwAAAAACgAKAAAGL0BAJcKRPECMT2AyajqbFsfzqUFMnYbMtVm4bEeLw7dB+BpCX4ogAVF0PBuMaBAEACH5BAVkACUALAAAAQAJAAgAAAYVwBJpSBwKi8QjkqRENovP5NI4JQUBADs="

	imgPendenteAtivacao = "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAQDAwMDAwQDAwQFBAMEBQcFBAQFBwgGBgcGBggKCAgICAgICggKCgsKCggNDQ4ODQ0SEhISEhQUFBQUFBQUFBT/2wBDAQUFBQgHCA8KCg8SDwwPEhYVFRUVFhYUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBT/wAARCAAKAAoDAREAAhEBAxEB/8QAFwAAAwEAAAAAAAAAAAAAAAAAAQQGCP/EAB4QAAICAgIDAAAAAAAAAAAAAAECAwQAERIhMVFh/8QAGAEAAgMAAAAAAAAAAAAAAAAAAAECAwT/xAAWEQEBAQAAAAAAAAAAAAAAAAAAARH/2gAMAwEAAhEDEQA/ANUXr15btlVszKBM4Ch260x+5pkJX0eT0qzu7l2hQsSeySoyqmFijSZ+RrQlmJLEouyT76w0G4lVY0VQAoUAAeNZEP/Z"
)

type clusterContrato struct {
	Token         string `json:"token"`
	Img           string `json:"img"`
	Cliente       string `json:"cliente"`
	Logradouro    string `json:"logradouro"`
	OrganizacaoID *int   `json:"organizacao_id,omitempty"`
}

type coordenadaGPS struct {
	Icon              string `json:"icon"`
	InfoWindowContent string `json:"infowindow_content"`
	Lat               string `json:"lat"`
	Lon               string `json:"lon"`
	Title             string `json:"title"`
	OrganizacaoID     int    `json:"organizacao_id"`
}

func HandlerRunCluster(instancia dominio.Instancia) error {
	tag := "run_cluster"
	logger.Inicio(tag, "Instancia %d: processando...", instancia.ID)

	db, err := banco.ConectarInstancia(
		instancia.EnvDBHost, instancia.EnvDBPort,
		instancia.EnvDBUser, instancia.EnvDBPass, instancia.EnvDBName,
	)
	if err != nil {
		return fmt.Errorf("falha ao conectar na instancia %d: %w", instancia.ID, err)
	}
	defer banco.FecharConexaoInstancia(db, tag)

	if err := processarClusterContratos(db); err != nil {
		return fmt.Errorf("erro em processarClusterContratos: %w", err)
	}

	if err := processarContratosCoordenadas(db); err != nil {
		return fmt.Errorf("erro em processarContratosCoordenadas: %w", err)
	}

	logger.Sucesso(tag, "Instancia %d concluida", instancia.ID)
	return nil
}

func processarClusterContratos(db *sql.DB) error {
	tag := "run_cluster"

	numOn, numOff, err := contarConexoes(db)
	if err != nil {
		return err
	}

	numContratos := numOn + numOff
	numBloqueados := contarBloqueados(db)
	numOSPendentes := contarOSPendentes(db)

	contratos, err := buscarClientesColmeia(db)
	if err != nil {
		return fmt.Errorf("erro ao buscar clientes colmeia: %w", err)
	}

	desconexoes, err := buscarDesconexoes(db)
	if err != nil {
		return fmt.Errorf("erro ao buscar desconexoes: %w", err)
	}

	var a []*clusterContrato

	if contratos == nil {
		a = append(a, nil)
	} else {
		for _, c := range contratos {
			item := montarItemCluster(c, desconexoes)
			a = append(a, item)
		}
	}

	clusterJSON, err := jsonEncodeIdentico(a)
	if err != nil {
		return fmt.Errorf("erro ao codificar JSON do cluster: %w", err)
	}

	if err := atualizarClusterContratos(db, clusterJSON, numOn, numOff, numContratos, numBloqueados, numOSPendentes); err != nil {
		return fmt.Errorf("erro ao atualizar sgp_webservices (cluster): %w", err)
	}

	logger.Info(tag, "Cluster: %d contratos, %d online, %d offline, %d bloqueados, %d OS pendentes",
		numContratos, numOn, numOff, numBloqueados, numOSPendentes)

	return nil
}

type linhaConexao struct {
	Conexao string `json:"conexao"`
}

func contarConexoes(db *sql.DB) (on, off int, err error) {
	query := `SELECT conexao FROM sgp_clientes_contratos
		WHERE conexao IN ('Online','Offline')
		AND status = 'Ativo'
		AND suspender_contrato = '0'
		AND ws_update_sequencia > 0`

	linhas, err := db.Query(query)
	if err != nil {
		return 0, 0, fmt.Errorf("erro ao contar conexoes: %w", err)
	}
	defer linhas.Close()

	for linhas.Next() {
		var c string
		if err := linhas.Scan(&c); err != nil {
			return 0, 0, fmt.Errorf("erro ao escanear conexao: %w", err)
		}
		if c == "Online" {
			on++
		} else if c == "Offline" {
			off++
		}
	}

	return on, off, linhas.Err()
}

func contarBloqueados(db *sql.DB) int {
	query := `SELECT id FROM sgp_clientes_contratos WHERE status = 'Bloqueado'`
	linhas, err := db.Query(query)
	if err != nil {
		return 0
	}
	defer linhas.Close()

	count := 0
	for linhas.Next() {
		count++
	}
	return count
}

func contarOSPendentes(db *sql.DB) int {
	query := `SELECT tb_os.id FROM sgp_clientes_contratos_os AS tb_os
		INNER JOIN sgp_clientes_contratos AS tb_contratos
			ON tb_contratos.token = tb_os.contrato_token
		INNER JOIN sgp_clientes_new AS tb_clientes
			ON tb_clientes.token = tb_contratos.cliente_token
		WHERE tb_os.status = 'Pendente'`

	linhas, err := db.Query(query)
	if err != nil {
		return 0
	}
	defer linhas.Close()

	count := 0
	for linhas.Next() {
		count++
	}
	return count
}

type linhaClienteColmeia struct {
	ID                              int
	Token                           string
	Tipo                            string
	PfNome                          sql.NullString
	PjRazaoSocial                   sql.NullString
	WsUpdateSequencia               int
	Logradouro                      sql.NullString
	LogradouroBairro                sql.NullString
	LogradouroNumero                sql.NullString
	ContratoToken                   string
	PppoeUser                       sql.NullString
	Status                          string
	PlanoMigracao                   sql.NullString
	ContratoID                      int
	LogradouroCoordenadasGPS        sql.NullString
	Conexao                         string
	DataHoraUltimaConexaoAtividade  sql.NullString
	OrganizacaoID                   sql.NullInt64
}

func buscarClientesColmeia(db *sql.DB) ([]linhaClienteColmeia, error) {
	query := `SELECT
		tb_clientes.id, tb_clientes.token, tb_clientes.tipo,
		tb_clientes.pf_nome, tb_clientes.pj_razao_social,
		tb_contratos.ws_update_sequencia, tb_contratos.logradouro,
		tb_contratos.logradouro_bairro, tb_contratos.logradouro_numero,
		tb_contratos.token AS contrato_token, tb_contratos.pppoe_user,
		tb_contratos.status, tb_contratos.plano_migracao,
		tb_contratos.id AS contrato_id,
		tb_contratos.logradouro_coordenadas_gps, tb_contratos.conexao,
		tb_contratos.data_hora_ultima_conexao_atividade,
		tb_contratos.organizacao_id
	FROM sgp_clientes_new AS tb_clientes
	INNER JOIN sgp_clientes_contratos AS tb_contratos
		ON tb_contratos.cliente_token = tb_clientes.token
	WHERE tb_contratos.status IN ('Ativo','Bloqueado')
		AND tb_contratos.suspender_contrato = '0'
	ORDER BY tb_contratos.logradouro ASC`

	linhas, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro na query clientes colmeia: %w", err)
	}
	defer linhas.Close()

	var resultado []linhaClienteColmeia
	for linhas.Next() {
		var l linhaClienteColmeia
		if err := linhas.Scan(
			&l.ID, &l.Token, &l.Tipo,
			&l.PfNome, &l.PjRazaoSocial,
			&l.WsUpdateSequencia, &l.Logradouro,
			&l.LogradouroBairro, &l.LogradouroNumero,
			&l.ContratoToken, &l.PppoeUser,
			&l.Status, &l.PlanoMigracao,
			&l.ContratoID,
			&l.LogradouroCoordenadasGPS, &l.Conexao,
			&l.DataHoraUltimaConexaoAtividade,
			&l.OrganizacaoID,
		); err != nil {
			return nil, fmt.Errorf("erro ao escanear cliente colmeia: %w", err)
		}
		resultado = append(resultado, l)
	}

	if len(resultado) == 0 {
		return nil, nil
	}

	return resultado, linhas.Err()
}

type linhaDesconexao struct {
	ContratoID int
	Contador2  int
}

func buscarDesconexoes(db *sql.DB) (map[int]int, error) {
	query := `SELECT sgp_clientes_contratos_desconexoes.contrato_id,
		sgp_clientes_contratos_desconexoes.contador2
	FROM sgp_clientes_contratos_desconexoes
	INNER JOIN sgp_clientes_contratos
		ON sgp_clientes_contratos_desconexoes.contrato_id = sgp_clientes_contratos.id
	WHERE sgp_clientes_contratos_desconexoes.data = CURDATE()
		AND sgp_clientes_contratos.suspender_contrato = '0'
		AND sgp_clientes_contratos.status IN ('Ativo','Bloqueado')`

	linhas, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()

	resultado := make(map[int]int)
	for linhas.Next() {
		var contratoID, contador2 int
		if err := linhas.Scan(&contratoID, &contador2); err != nil {
			return nil, err
		}
		resultado[contratoID] = contador2
	}

	return resultado, linhas.Err()
}

func montarItemCluster(c linhaClienteColmeia, desconexoes map[int]int) *clusterContrato {
	nomeCliente := ""
	if c.PfNome.Valid {
		nomeCliente = c.PfNome.String
	}
	if c.PjRazaoSocial.Valid && c.PjRazaoSocial.String != "" && nomeCliente == "" {
		nomeCliente = c.PjRazaoSocial.String
	}

	logradouro := ""
	if c.Logradouro.Valid {
		logradouro = c.Logradouro.String
	}
	if c.LogradouroNumero.Valid {
		logradouro += ", " + c.LogradouroNumero.String
	}
	if c.LogradouroBairro.Valid {
		logradouro += ", " + c.LogradouroBairro.String
	}

	item := &clusterContrato{
		Token:    c.ContratoToken,
		Cliente:  nomeCliente,
		Logradouro: logradouro,
	}

	if d, ok := desconexoes[c.ContratoID]; ok && d >= 7 {
		item.Img = imgDesconexao
		if c.OrganizacaoID.Valid {
			orgID := int(c.OrganizacaoID.Int64)
			item.OrganizacaoID = &orgID
		}
	} else if c.Status == "Bloqueado" {
		item.Img = imgBloqueado
		if c.OrganizacaoID.Valid {
			orgID := int(c.OrganizacaoID.Int64)
			item.OrganizacaoID = &orgID
		}
	} else if c.WsUpdateSequencia == 0 {
		item.Img = imgPendenteAtivacao
	} else {
		if c.Conexao == "Offline" {
			item.Img = imgOffline
		} else {
			item.Img = imgOnline
		}
		if c.OrganizacaoID.Valid {
			orgID := int(c.OrganizacaoID.Int64)
			item.OrganizacaoID = &orgID
		}
	}

	return item
}

func atualizarClusterContratos(db *sql.DB, clusterJSON string, numOn, numOff, numContratos, numBloqueados, numOSPendentes int) error {
	query := `UPDATE sgp_webservices SET
		cluster_contratos_status = ?,
		num_online = ?,
		num_offline = ?,
		num_contratos = ?,
		num_contratos_bloqueados = ?,
		num_os_pendentes = ?
	WHERE id = 0`

	_, err := db.Exec(query, clusterJSON, numOn, numOff, numContratos, numBloqueados, numOSPendentes)
	if err != nil {
		return fmt.Errorf("erro ao atualizar cluster_contratos_status: %w", err)
	}

	return nil
}

type linhaClienteCoordenada struct {
	ID                              int
	Token                           string
	Tipo                            string
	PfNome                          sql.NullString
	PjRazaoSocial                   sql.NullString
	Logradouro                      sql.NullString
	LogradouroBairro                sql.NullString
	LogradouroNumero                sql.NullString
	ContratoToken                   string
	PppoeUser                       sql.NullString
	Status                          string
	PlanoMigracao                   sql.NullString
	ContratoID                      int
	LogradouroCoordenadasGPS        string
	Conexao                         string
	DataHoraUltimaConexaoAtividade  sql.NullString
	OrganizacaoID                   sql.NullInt64
}

func processarContratosCoordenadas(db *sql.DB) error {
	tag := "run_cluster"

	clientes, err := buscarClientesCoordenadas(db)
	if err != nil {
		return fmt.Errorf("erro ao buscar clientes coordenadas: %w", err)
	}

	desconexoes, err := buscarDesconexoes(db)
	if err != nil {
		return fmt.Errorf("erro ao buscar desconexoes: %w", err)
	}

	var gps []coordenadaGPS

	if clientes != nil {
		for _, c := range clientes {
			item, incluir := montarItemCoordenada(c, desconexoes)
			if incluir {
				gps = append(gps, item)
			}
		}
	}

	if gps == nil {
		gps = []coordenadaGPS{}
	}

	coordenadasJSON, err := jsonEncodeIdentico(gps)
	if err != nil {
		return fmt.Errorf("erro ao codificar JSON das coordenadas: %w", err)
	}

	if err := atualizarCoordenadas(db, coordenadasJSON); err != nil {
		return fmt.Errorf("erro ao atualizar coordenadas_contratos_status: %w", err)
	}

	logger.Info(tag, "Coordenadas: %d marcadores processados", len(gps))

	return nil
}

func buscarClientesCoordenadas(db *sql.DB) ([]linhaClienteCoordenada, error) {
	query := `SELECT
		tb_clientes.id, tb_clientes.token, tb_clientes.tipo,
		tb_clientes.pf_nome, tb_clientes.pj_razao_social,
		tb_contratos.logradouro, tb_contratos.logradouro_bairro,
		tb_contratos.logradouro_numero,
		tb_contratos.token AS contrato_token, tb_contratos.pppoe_user,
		tb_contratos.status, tb_contratos.plano_migracao,
		tb_contratos.id AS contrato_id,
		tb_contratos.logradouro_coordenadas_gps, tb_contratos.conexao,
		tb_contratos.data_hora_ultima_conexao_atividade,
		tb_contratos.organizacao_id
	FROM sgp_clientes_new AS tb_clientes
	INNER JOIN sgp_clientes_contratos AS tb_contratos
		ON tb_contratos.cliente_token = tb_clientes.token
	WHERE tb_contratos.status = 'Ativo'
		AND tb_contratos.logradouro_coordenadas_gps IS NOT NULL
	ORDER BY tb_clientes.pf_nome ASC, tb_clientes.pj_razao_social ASC`

	linhas, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro na query clientes coordenadas: %w", err)
	}
	defer linhas.Close()

	var resultado []linhaClienteCoordenada
	for linhas.Next() {
		var l linhaClienteCoordenada
		if err := linhas.Scan(
			&l.ID, &l.Token, &l.Tipo,
			&l.PfNome, &l.PjRazaoSocial,
			&l.Logradouro, &l.LogradouroBairro,
			&l.LogradouroNumero,
			&l.ContratoToken, &l.PppoeUser,
			&l.Status, &l.PlanoMigracao,
			&l.ContratoID,
			&l.LogradouroCoordenadasGPS, &l.Conexao,
			&l.DataHoraUltimaConexaoAtividade,
			&l.OrganizacaoID,
		); err != nil {
			return nil, fmt.Errorf("erro ao escanear cliente coordenada: %w", err)
		}
		resultado = append(resultado, l)
	}

	return resultado, linhas.Err()
}

func montarItemCoordenada(c linhaClienteCoordenada, desconexoes map[int]int) (coordenadaGPS, bool) {
	nomeCliente := ""
	if c.PfNome.Valid {
		nomeCliente = c.PfNome.String
	}
	if c.PjRazaoSocial.Valid && c.PjRazaoSocial.String != "" && nomeCliente == "" {
		nomeCliente = c.PjRazaoSocial.String
	}

	logradouro := ""
	if c.Logradouro.Valid {
		logradouro = c.Logradouro.String
	}
	if c.LogradouroNumero.Valid {
		logradouro += ", " + c.LogradouroNumero.String
	}
	if c.LogradouroBairro.Valid {
		logradouro += " - " + c.LogradouroBairro.String
	}

	var ultimaAtividade time.Time
	if c.DataHoraUltimaConexaoAtividade.Valid {
		parsed, err := time.Parse("2006-01-02 15:04:05", c.DataHoraUltimaConexaoAtividade.String)
		if err == nil {
			ultimaAtividade = parsed
		}
	}

	diasInativo := int(time.Since(ultimaAtividade).Hours() / 24)

	if diasInativo >= 5 {
		return coordenadaGPS{}, false
	}

	partes := strings.Split(c.LogradouroCoordenadasGPS, ",")
	lat := ""
	lon := ""
	if len(partes) > 0 {
		lat = strings.TrimSpace(partes[0])
	}
	if len(partes) > 1 {
		lon = strings.TrimSpace(partes[1])
	}

	icon := ""
	if d, ok := desconexoes[c.ContratoID]; ok && d >= 7 {
		if c.Conexao == "Online" {
			icon = "/assets/ISP/img/ico/exclamacao_laranja.png"
		} else {
			icon = "/assets/ISP/img/ico/exclamacao_vermelho.png"
		}
	} else if c.Conexao == "Online" {
		icon = "/assets/ISP/img/ico/Map-Marker-Flag-4-Right-Chartreuse-icon.png"
	} else {
		if diasInativo >= 1 && diasInativo <= 5 {
			icon = "/assets/ISP/img/ico/triangulo_amarelo.png"
		} else {
			icon = "/assets/ISP/img/ico/Map-Marker-Flag-4-Right-Pink-icon.png"
		}
	}

	orgID := 0
	if c.OrganizacaoID.Valid {
		orgID = int(c.OrganizacaoID.Int64)
	}

	marker := coordenadaGPS{
		Icon:              icon,
		InfoWindowContent: fmt.Sprintf("<strong>%s</strong><br><strong>Endereço:</strong> %s", nomeCliente, logradouro),
		Lat:               lat,
		Lon:               lon,
		Title:             nomeCliente,
		OrganizacaoID:     orgID,
	}

	return marker, true
}

func atualizarCoordenadas(db *sql.DB, coordenadasJSON string) error {
	query := `UPDATE sgp_webservices SET
		coordenadas_contratos_status = ?
	WHERE id = 0`

	_, err := db.Exec(query, coordenadasJSON)
	if err != nil {
		return fmt.Errorf("erro ao atualizar coordenadas_contratos_status: %w", err)
	}

	return nil
}

func jsonEncodeIdentico(v interface{}) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}
