package repositorio

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Tipos exportados
// ---------------------------------------------------------------------------

// LinhaClienteColmeia representa um cliente apto para agrupacao colmeia.
type LinhaClienteColmeia struct {
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

// LinhaDesconexao representa uma linha de desconexao do contrato.
type LinhaDesconexao struct {
	ContratoID int
	Contador2  int
}

// LinhaClienteCoordenada representa um cliente com coordenadas geograficas.
type LinhaClienteCoordenada struct {
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

// ---------------------------------------------------------------------------
// Contagem de conexoes do cluster
// ---------------------------------------------------------------------------

// ContarConexoesCluster retorna o total de conexoes Online e Offline na
// tabela sgp_clientes_contratos para contratos ativos nao suspensos.
// Extraido de contarConexoes (run_cluster.go).
func ContarConexoesCluster(db *sql.DB) (on, off int, err error) {
	query := `SELECT conexao FROM sgp_clientes_contratos
		WHERE conexao IN ('Online','Offline')
		AND status = 'Ativo'
		AND suspender_contrato = '0'
		AND ws_update_sequencia > 0`

	linhas, err := db.Query(query)
	if err != nil {
		return 0, 0, fmt.Errorf("contar conexoes cluster: %w", err)
	}
	defer linhas.Close()

	for linhas.Next() {
		var c string
		if err := linhas.Scan(&c); err != nil {
			return 0, 0, fmt.Errorf("contar conexoes cluster: erro ao escanear: %w", err)
		}
		if c == "Online" {
			on++
		} else if c == "Offline" {
			off++
		}
	}

	return on, off, linhas.Err()
}

// ---------------------------------------------------------------------------
// Contagem de bloqueados
// ---------------------------------------------------------------------------

// ContarBloqueados retorna o total de contratos com status 'Bloqueado'.
// Extraido de contarBloqueados (run_cluster.go).
func ContarBloqueados(db *sql.DB) int {
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

// ---------------------------------------------------------------------------
// Contagem de OS pendentes
// ---------------------------------------------------------------------------

// ContarOSPendentes retorna o total de ordens de servico com status
// 'Pendente'. Extraido de contarOSPendentes (run_cluster.go).
func ContarOSPendentes(db *sql.DB) int {
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

// ---------------------------------------------------------------------------
// Busca de clientes colmeia
// ---------------------------------------------------------------------------

// BuscarClientesColmeia retorna clientes aptos para agrupamento colmeia
// (contratos Ativos/Bloqueados nao suspensos), ordenados por logradouro.
// Extraido de buscarClientesColmeia (run_cluster.go).
func BuscarClientesColmeia(db *sql.DB) ([]LinhaClienteColmeia, error) {
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
		return nil, fmt.Errorf("buscar clientes colmeia: %w", err)
	}
	defer linhas.Close()

	var resultado []LinhaClienteColmeia
	for linhas.Next() {
		var l LinhaClienteColmeia
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
			return nil, fmt.Errorf("buscar clientes colmeia: erro ao escanear: %w", err)
		}
		resultado = append(resultado, l)
	}

	if len(resultado) == 0 {
		return nil, nil
	}

	return resultado, linhas.Err()
}

// ---------------------------------------------------------------------------
// Busca de desconexoes
// ---------------------------------------------------------------------------

// BuscarDesconexoes retorna um mapa de contrato_id -> contador2 para
// contratos com desconexao no dia atual.
// Extraido de buscarDesconexoes (run_cluster.go).
func BuscarDesconexoes(db *sql.DB) (map[int]int, error) {
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
		return nil, fmt.Errorf("buscar desconexoes: %w", err)
	}
	defer linhas.Close()

	resultado := make(map[int]int)
	for linhas.Next() {
		var contratoID, contador2 int
		if err := linhas.Scan(&contratoID, &contador2); err != nil {
			return nil, fmt.Errorf("buscar desconexoes: erro ao escanear: %w", err)
		}
		resultado[contratoID] = contador2
	}

	return resultado, linhas.Err()
}

// ---------------------------------------------------------------------------
// Atualizacao de cluster contratos
// ---------------------------------------------------------------------------

// AtualizarClusterContratos atualiza os dados de cluster na tabela
// sgp_webservices.
// Extraido de atualizarClusterContratos (run_cluster.go).
func AtualizarClusterContratos(db *sql.DB, clusterJSON string, numOn, numOff, numContratos, numBloqueados, numOSPendentes int) error {
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
		return fmt.Errorf("atualizar cluster contratos: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Busca de clientes com coordenadas
// ---------------------------------------------------------------------------

// BuscarClientesCoordenadas retorna clientes Ativos que possuem
// coordenadas GPS preenchidas, ordenados por nome / razao social.
// Extraido de buscarClientesCoordenadas (run_cluster.go).
func BuscarClientesCoordenadas(db *sql.DB) ([]LinhaClienteCoordenada, error) {
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
		return nil, fmt.Errorf("buscar clientes coordenadas: %w", err)
	}
	defer linhas.Close()

	var resultado []LinhaClienteCoordenada
	for linhas.Next() {
		var l LinhaClienteCoordenada
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
			return nil, fmt.Errorf("buscar clientes coordenadas: erro ao escanear: %w", err)
		}
		resultado = append(resultado, l)
	}

	return resultado, linhas.Err()
}

// ---------------------------------------------------------------------------
// Atualizacao de coordenadas
// ---------------------------------------------------------------------------

// AtualizarCoordenadas atualiza os dados de coordenadas de contratos na
// tabela sgp_webservices.
// Extraido de atualizarCoordenadas (run_cluster.go).
func AtualizarCoordenadas(db *sql.DB, coordenadasJSON string) error {
	query := `UPDATE sgp_webservices SET
		coordenadas_contratos_status = ?
	WHERE id = 0`

	_, err := db.Exec(query, coordenadasJSON)
	if err != nil {
		return fmt.Errorf("atualizar coordenadas: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// JSON encoding (mesmo comportamento do encoding/json padrao sem escapar
// HTML, igual ao original)
// ---------------------------------------------------------------------------

// JSONEncodeIdentico codifica um valor como JSON sem escapar caracteres
// HTML e removendo o newline final.
// Extraido de jsonEncodeIdentico (run_cluster.go).
func JSONEncodeIdentico(v interface{}) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return "", fmt.Errorf("json encode identico: %w", err)
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}
