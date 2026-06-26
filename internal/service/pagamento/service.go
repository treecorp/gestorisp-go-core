package pagamento

import (
	"database/sql"

	"gestor/internal/entity"
	"gestor/internal/lib/iugu"
	"gestor/internal/repositorio"
)

// Queryer é a interface para operações de consulta (QueryRow), satisfeita
// por *sql.DB e *sql.Tx. Reexportada de internal/repositorio para
// conveniência do pacote.
type Queryer = repositorio.Queryer

// FaturaRepo define as operações de persistência relacionadas a faturas
// e caixa.
type FaturaRepo interface {
	// BuscarFaturaPorToken busca uma fatura na tabela sgp_clientes_faturas
	// pelo token (external_reference do Iugu).
	BuscarFaturaPorToken(db *sql.DB, token string) (*entity.Fatura, error)

	// BuscarGatewayToken retorna o token Iugu do gateway de pagamento.
	BuscarGatewayToken(q Queryer, gatewayID int64) (string, error)

	// AtualizarStatusFatura atualiza o status da fatura para "Pago",
	// registrando valor_pago, data_pagamento, origem, protocolo_baixa
	// e demais campos de baixa.
	AtualizarStatusFatura(tx *sql.Tx, faturaID int, valorPago, dataPagto, origem, dataHora, protocoloBaixa string) error

	// BuscarSaldoCaixa retorna o saldo atual de um caixa pelo ID.
	BuscarSaldoCaixa(tx *sql.Tx, caixaID int) (int, error)

	// AtualizarSaldoCaixa atualiza o saldo de um caixa.
	AtualizarSaldoCaixa(tx *sql.Tx, caixaID, novoSaldo int) error

	// SomarSaldosCaixas retorna a soma dos saldos de todos os caixas.
	SomarSaldosCaixas(tx *sql.Tx) (int, error)

	// InserirFluxoCaixa insere um registro na tabela gisp_fluxos_caixas.
	InserirFluxoCaixa(tx *sql.Tx, saldoGlobal, saldoAnterior, saldoAtual, valor int, dataHora, token, protocolo, descricao string) error
}

// ContratoRepo define as operações de persistência relacionadas a contratos.
type ContratoRepo interface {
	// BuscarContratoPorID busca um contrato pelo ID com LEFT JOIN
	// sgp_clientes_new para obter o nome do cliente.
	BuscarContratoPorID(q Queryer, contratoID int) (*entity.Contrato, error)

	// DesbloquearContrato atualiza o status do contrato para "Ativo".
	DesbloquearContrato(tx *sql.Tx, contratoID int) error

	// InserirProtocolo insere um registro na tabela
	// sgp_clientes_contratos_protocolos.
	InserirProtocolo(tx *sql.Tx, token, contratoToken string, contratoID int, protocolo, dataHora, descricao, titulo, dadosAntigos, dadosNovos string) error

	// RemoverRadReplyCorte remove registros de bloqueio (pgcorte) na tabela
	// radreply para o contrato informado.
	RemoverRadReplyCorte(tx *sql.Tx, contratoID int) error
}

// GatilhoRepo define as operações de persistência relacionadas a gatilhos
// Iugu (gisp_iugu_gatilhos) e dados de fatura JSON.
type GatilhoRepo interface {
	// InserirGatilhoCompleto insere o registro inicial em gisp_iugu_gatilhos
	// com todos os campos disponíveis, antes da transação. Usa ON DUPLICATE
	// KEY para idempotência.
	InserirGatilhoCompleto(db *sql.DB, iuguFaturaID, accountID, externalRef, status, event, dadosJSON string) error

	// VerificarGatilhoProcessado verifica se o gatilho já foi processado
	// (gisp_exec = '1') para o event 'invoice.status_changed'.
	VerificarGatilhoProcessado(db *sql.DB, iuguFaturaID string) (bool, error)

	// VerificarGatilhoExternalProcessado verifica se o gatilho já foi
	// processado (gisp_exec = '1') para status 'externally_paid'.
	VerificarGatilhoExternalProcessado(db *sql.DB, iuguFaturaID string) (bool, error)

	// InserirGatilho registra um gatilho com status "Processando" dentro
	// de uma transação. Equivalente ao marcarProcessando do código original.
	InserirGatilho(tx *sql.Tx, iuguFaturaID, statusEsperado string) error

	// MarcarProcessado atualiza o gatilho como processado (gisp_exec='1',
	// gisp_exec_status='Processado') dentro de uma transação.
	MarcarProcessado(tx *sql.Tx, iuguFaturaID, status, protocolo string) error

	// MarcarErroGatilho registra erro no processamento do gatilho fora de
	// transação.
	MarcarErroGatilho(db *sql.DB, iuguFaturaID, status, codErro, msg string) error

	// SalvarFaturaJSON insere ou atualiza os dados da fatura Iugu em
	// gisp_iugu_faturas_json.
	SalvarFaturaJSON(tx *sql.Tx, iuguFaturaID string, fatura *iugu.FaturaIugu, dadosJSON string) error
}

// PopRepo define as operações de persistência relacionadas a POPs.
type PopRepo interface {
	// BuscarPopsOperacionais retorna todos os POPs com status "OPERACIONAL".
	BuscarPopsOperacionais(db *sql.DB) ([]entity.Pop, error)
}

// ClienteIugu define a interface para consulta de faturas na API Iugu.
// Permite injeção de dependência para testes sem chamadas HTTP reais.
type ClienteIugu interface {
	// ConsultarFatura busca os dados de uma fatura na API Iugu pelo seu
	// identificador.
	ConsultarFatura(faturaID string) (*iugu.FaturaIugu, error)
}

// Repositorios agrupa as interfaces de repositório necessárias para o
// service de pagamento, facilitando injeção de dependência e testes.
type Repositorios struct {
	Fatura   FaturaRepo
	Contrato ContratoRepo
	Gatilho  GatilhoRepo
	Pop      PopRepo
}
