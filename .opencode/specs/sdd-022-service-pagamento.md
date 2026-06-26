# SDD-022 — Service de Pagamento

**Status:** Pendente
**Autor:** Knowledge Engineer
**Prioridade:** Alta
**Dependencias:** SDD-018 (`entity.Instancia`, `entity.Contrato`, `entity.Fatura`), SDD-019 (`lib/iugu`), SDD-020 (`repositorio`)

## 1. Objetivo

Extrair regras de negócio de `processar.go` para o pacote `internal/service/pagamento/`, removendo acoplamento direto com `*sql.DB` e promovendo um design orientado a serviços com dependências injetadas via interfaces ou structs.

## 2. Escopo

### 2.1 internal/service/pagamento/processar.go

Função orquestradora principal:

```go
// ProcessarPagamento orquestra o fluxo completo de processamento de um
// pagamento Iugu: consulta fatura, identifica contrato, executa baixa,
// atualiza status e dispara desbloqueio se necessário.
func ProcessarPagamento(
    repo *Repositorios,
    iugu *lib.ClienteIugu,
    instancia entity.Instancia,
    data map[string]string,
    faturaID string,
    status string,
    event string,
) (*ResultadoBaixa, error)
```

Funções privadas:

```go
// processarIuguDireto processa pagamento via fluxo Iugu direto.
func processarIuguDireto(...) (*ResultadoBaixa, error)

// processarExternal processa pagamento via fluxo externo.
func processarExternal(...) (*ResultadoBaixa, error)

// externalRef gera a referência externa para a fatura.
func externalRef(fatura entity.Fatura) string
```

### 2.2 internal/service/pagamento/baixa.go

```go
// ExecutarBaixa executa a lógica de baixa contábil do pagamento.
func ExecutarBaixa(...) error

// LancarCaixa registra o lançamento no caixa.
func LancarCaixa(...) error

// CriarProtocoloBaixa gera um novo protocolo de baixa.
func CriarProtocoloBaixa() string

// ResultadoBaixa contém os dados retornados após processar baixa.
type ResultadoBaixa struct {
    Sucesso    bool
    Protocolo  string
    Mensagem   string
    // demais campos relevantes
}
```

### 2.3 internal/service/pagamento/contrato.go

```go
// DesbloquearContrato executa a regra de negócio de desbloqueio de
// contrato após pagamento confirmado.
func DesbloquearContrato(...) error
```

### 2.4 internal/service/pagamento/origem.go

```go
// OrigemPagamento retorna o código de origem (para caixa) baseado no
// método de pagamento.
func OrigemPagamento(metodo string) string

// codigosOrigem mapeia método de pagamento para código de origem.
var codigosOrigem map[string]string
```

### 2.5 Struct Repositorios

```go
// Repositorios agrupa as interfaces de repositório necessárias para o
// service de pagamento, facilitando injeção de dependência e testes.
type Repositorios struct {
    Fatura    FaturaRepo
    Contrato  ContratoRepo
    Gatilho   GatilhoRepo
}

type FaturaRepo interface {
    BuscarFaturaPorToken(db *sql.DB, token string) (*entity.Fatura, error)
    AtualizarStatusFatura(tx *sql.Tx, faturaID int, status, protocolo string) error
}

type ContratoRepo interface {
    BuscarContratoPorID(db *sql.DB, contratoID int) (*entity.Contrato, error)
    DesbloquearContrato(tx *sql.Tx, contratoID int) error
}

type GatilhoRepo interface {
    InserirGatilho(tx *sql.Tx, iuguFaturaID, statusEsperado string) error
    MarcarProcessado(tx *sql.Tx, iuguFaturaID, status, protocolo string) error
    MarcarErroGatilho(db *sql.DB, iuguFaturaID, status, codErro, msg string) error
}
```

## 3. Dependências

- `database/sql` — apenas para interfaces (não acoplamento direto)
- `internal/entity` — structs de domínio
- `internal/lib/iugu` — cliente Iugu
- `internal/helpers` — funções auxiliares

## 4. Arquivos a criar

| Arquivo | Ação |
|---------|------|
| `internal/service/pagamento/processar.go` | Criar — orquestrador |
| `internal/service/pagamento/baixa.go` | Criar — lógica de baixa |
| `internal/service/pagamento/contrato.go` | Criar — desbloqueio |
| `internal/service/pagamento/origem.go` | Criar — origem pagamento |
| `internal/service/pagamento/service.go` | Criar — interfaces + Repositorios struct |
| `internal/service/pagamento/doc.go` | Criar — package-level doc |

## 5. Documentação

Toda função exportada deve ter **doc comment em português**:

```go
// Package pagamento implementa as regras de negócio para processamento
// de pagamentos Iugu, incluindo baixa contábil, desbloqueio de contratos
// e geração de protocolos.
package pagamento
```

```go
// ProcessarPagamento é o orquestrador principal do fluxo de pagamento.
// Recebe os dados do webhook Iugu, consulta repositórios, aplica regras
// de negócio e retorna o resultado da baixa.
func ProcessarPagamento(...) (*ResultadoBaixa, error)
```

## 6. Critérios de Aceite

- [ ] `internal/service/pagamento/` compila sem erros
- [ ] Nenhuma função do service recebe `*sql.DB` diretamente (usa interfaces)
- [ ] Lógica extraída de `processar.go` mantém o mesmo comportamento
- [ ] Todas as funções possuem doc comment em português
- [ ] `doc.go` presente no pacote
- [ ] Código original `processar.go` permanece intacto
