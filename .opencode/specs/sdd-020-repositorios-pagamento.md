# SDD-020 — Repositórios de Pagamento

**Status:** Pendente
**Autor:** Knowledge Engineer
**Prioridade:** Alta
**Dependencias:** SDD-018 (usa `entity.Contrato`, `entity.Fatura`)

## 1. Objetivo

Extrair todas as queries SQL relacionadas a pagamento de `processar.go` e `listar_clientes_vencidos.go` para o pacote `internal/repositorio/`, centralizando o acesso a dados e removendo SQL espalhado pelos arquivos de lógica.

## 2. Escopo

### 2.1 internal/repositorio/fatura_repo.go

```go
// BuscarFaturaPorToken busca uma fatura pelo token da instância e
// identificador da fatura Iugu.
func BuscarFaturaPorToken(db *sql.DB, token string) (*entity.Fatura, error)

// AtualizarStatusFatura atualiza o status e protocolo de uma fatura
// dentro de uma transação.
func AtualizarStatusFatura(tx *sql.Tx, faturaID int, status, protocolo string) error
```

Extraído de:
- `processar.go`: query `SELECT id, token, fatura_id, ... FROM faturas WHERE token = ? AND fatura_id = ?`
- `processar.go`: query `UPDATE faturas SET status = ?, protocolo = ? WHERE id = ?`

### 2.2 internal/repositorio/contrato_repo.go

```go
// BuscarContratoPorID busca um contrato pelo seu identificador.
// Extraído da função buscarContrato em processar.go.
func BuscarContratoPorID(db *sql.DB, contratoID int) (*entity.Contrato, error)

// DesbloquearContrato altera o status do contrato para desbloqueado
// dentro de uma transação. Extraído de desbloquearContratoDB.
func DesbloquearContrato(tx *sql.Tx, contratoID int) error
```

Extraído de:
- `processar.go`: função `buscarContrato`
- `processar.go`: parte DB de `desbloquearContratoDB`

### 2.3 internal/repositorio/gatilho_repo.go

```go
// InserirGatilho insere um registro de gatilho de fatura Iugu para
// processamento posterior.
func InserirGatilho(tx *sql.Tx, iuguFaturaID, statusEsperado string) error

// MarcarProcessado marca um gatilho como processado com sucesso.
func MarcarProcessado(tx *sql.Tx, iuguFaturaID, status, protocolo string) error

// MarcarErroGatilho registra erro no processamento de um gatilho.
func MarcarErroGatilho(db *sql.DB, iuguFaturaID, status, codErro, msg string) error
```

Extraído de:
- `processar.go`: queries de manipulação da tabela `iugu_gatilhos` ou equivalente

### 2.4 internal/repositorio/bloqueio_repo.go

```go
// BuscarFaturasVencidas retorna todas as faturas vencidas não pagas.
// Extraído de listar_clientes_vencidos.go.
func BuscarFaturasVencidas(db *sql.DB) ([]entity.Fatura, error)

// LerDesbloqueioConfianca retorna configuração de desbloqueio por
// confiança para um contrato.
func LerDesbloqueioConfianca(db *sql.DB, contratoID int) (*DesbloqueioConfianca, error)

// LerDiasBloqueio retorna a configuração global de dias para bloqueio.
func LerDiasBloqueio(db *sql.DB) int
```

Extraído de:
- `listar_clientes_vencidos.go`: query `SELECT faturas.*, contratos.* FROM faturas JOIN contratos ...`
- `listar_clientes_vencidos.go`: queries auxiliares de configuração

## 3. Dependências

- `database/sql` — interface de banco
- `internal/entity` — structs `Contrato`, `Fatura`, `Pop`
- `internal/infra/banco` — pool de conexões (não importado diretamente, apenas `*sql.DB`)

## 4. Arquivos a criar

| Arquivo | Ação |
|---------|------|
| `internal/repositorio/fatura_repo.go` | Criar — queries de fatura |
| `internal/repositorio/contrato_repo.go` | Criar — queries de contrato |
| `internal/repositorio/gatilho_repo.go` | Criar — queries de gatilho |
| `internal/repositorio/bloqueio_repo.go` | Criar — queries de bloqueio |
| `internal/repositorio/doc.go` | Criar — package-level doc |

## 5. Documentação

Toda função deve ter **doc comment em português** descrevendo a query executada e os parâmetros:

```go
// BuscarFaturaPorToken consulta a tabela faturas filtrando pelo token
// da instância. Retorna nil se não encontrar registro.
func BuscarFaturaPorToken(db *sql.DB, token string) (*entity.Fatura, error)
```

Package `doc.go`:

```go
// Package repositorio implementa o padrão Repository para acesso a
// dados do sistema. Cada arquivo agrupa operações relacionadas a uma
// entidade (fatura, contrato, gatilho, bloqueio, etc.).
//
// Todas as funções recebem *sql.DB ou *sql.Tx explicitamente,
// permitindo controle transacional pelo chamador.
package repositorio
```

## 6. Estrutura de dados auxiliares

Tipos internos que podem ser necessários:

```go
// DesbloqueioConfianca representa a configuração de desbloqueio
// automático por confiança para um contrato.
type DesbloqueioConfianca struct {
    Ativo bool
    Dias  int
}
```

## 7. Critérios de Aceite

- [ ] `internal/repositorio/` compila sem erros com `go build ./...`
- [ ] Queries extraídas produzem exatamente o mesmo SQL que o código original
- [ ] Todas as funções possuem doc comment em português
- [ ] `doc.go` presente no pacote
