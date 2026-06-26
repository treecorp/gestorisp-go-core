# SDD-018 — Entity + Helpers

**Status:** Pendente
**Autor:** Knowledge Engineer
**Prioridade:** Alta
**Dependencias:** Nenhuma (criação de novos pacotes)

## 1. Objetivo

Criar os pacotes `internal/entity/` (tipos de domínio enriquecidos) e `internal/helpers/` (funções puras extraídas de diversos pacotes), promovendo reuso e separação de responsabilidades.

## 2. Escopo

### 2.1 internal/entity/instancia.go

Mover struct `Instancia` de `internal/dominio/instancia.go`:

```go
type Instancia struct {
    ID         int
    Token      string
    EnvDBHost  string
    EnvDBPort  string
    EnvDBUser  string
    EnvDBPass  string
    EnvDBName  string
}
```

Adicionar método:
- `GetID() int` — retorna `i.ID`

### 2.2 internal/entity/contrato.go

Struct `Contrato` baseada em `contratoRow`:

```go
type Contrato struct {
    ID          int
    ClienteID   int
    ClienteToken string
    Nome        string
    PPPoEUser   string
    PopIPv4     string
    PopPort     string
    PopUser     string
    PopPass     string
    Status      string
    // demais campos conforme contratoRow
}
```

Métodos:
- `EstaBloqueado() bool` — verifica se status indica bloqueio
- `Desbloquear()` — altera status para desbloqueado (apenas em memória)

### 2.3 internal/entity/fatura.go

Struct `Fatura` baseada em `faturaRow`:

```go
type Fatura struct {
    ID              int
    Token           string
    FaturaID        string
    Valor           string
    ValorPago       string
    DataVencimento  string
    DataPagamento   string
    Status          string
    // demais campos conforme faturaRow
}
```

Métodos:
- `EstaPaga() bool` — verifica status de pagamento
- `CalcularDiasAtraso() int` — calcula dias entre vencimento e hoje

### 2.4 internal/entity/pagamento.go

Struct `MensagemPagamentoIugu` movida de `dominio/pagamento.go`:

```go
type MensagemPagamentoIugu struct {
    // campos existentes
}
```

### 2.5 internal/entity/desconexao.go

Struct `MensagemDesconexaoContrato` movida de `dominio/desconexao.go`:

```go
type MensagemDesconexaoContrato struct {
    // campos existentes
}
```

Método:
- `Expirada() bool` — verifica se mensagem expirou

### 2.6 internal/entity/pop.go

Struct `Pop` com campos SQL, movida de `dominio/pop.go`:

```go
type Pop struct {
    // campos conforme dominio/pop.go
}
```

### 2.7 internal/helpers/moeda.go

Funções puras extraídas de `processar.go`:

```go
func FormatarMoeda(valor string) string
func LimparNumero(valor string) string
```

### 2.8 internal/helpers/string.go

```go
func Truncate(s string, max int) string
```

Extraída de `processar.go` e `gateway/iugu_webhook.go`.

### 2.9 internal/helpers/protocolo.go

```go
var idCounter int  // estado global

func GerarProtocolo(min, max int) int
```

Extraído de `processar.go`.

### 2.10 internal/helpers/data.go

```go
func ExtrairData(dt string) string
func ExtrairHora(dt string) string
```

Extraídas de `cron_1.go`.

### 2.11 internal/helpers/gerar_token.go

```go
func GerarToken() string
```

Extraída de `cron_1.go`, usando `crypto/rand` como base.

## 3. Dependências

Nenhuma. Pacotes `entity` e `helpers` serão importados pelos demais pacotes.

## 4. Arquivos a criar

| Arquivo | Ação |
|---------|------|
| `internal/entity/instancia.go` | Criar — struct + GetID |
| `internal/entity/contrato.go` | Criar — struct + métodos |
| `internal/entity/fatura.go` | Criar — struct + métodos |
| `internal/entity/pagamento.go` | Criar — struct |
| `internal/entity/desconexao.go` | Criar — struct + Expirada |
| `internal/entity/pop.go` | Criar — struct |
| `internal/entity/doc.go` | Criar — package-level doc |
| `internal/helpers/moeda.go` | Criar — funções puras |
| `internal/helpers/string.go` | Criar — Truncate |
| `internal/helpers/protocolo.go` | Criar — GerarProtocolo |
| `internal/helpers/data.go` | Criar — ExtrairData, ExtrairHora |
| `internal/helpers/gerar_token.go` | Criar — GerarToken |
| `internal/helpers/doc.go` | Criar — package-level doc |
| `internal/helpers/moeda_test.go` | Criar — testes unitários |
| `internal/helpers/string_test.go` | Criar — testes unitários |
| `internal/helpers/protocolo_test.go` | Criar — testes unitários |
| `internal/helpers/data_test.go` | Criar — testes unitários |
| `internal/helpers/gerar_token_test.go` | Criar — testes unitários |

## 5. Documentação

Toda função e tipo deve ter **doc comment em português** no formato:

```go
// Instancia representa uma instância GISP com suas credenciais de banco.
type Instancia struct { ... }

// GetID retorna o identificador numérico da instância.
func (i Instancia) GetID() int { ... }
```

Cada package deve conter um arquivo `doc.go` com comentário de package:

```go
// Package entity define os tipos de domínio enriquecidos do sistema.
//
// As structs aqui definidas carregam comportamentos (métodos) além dos
// dados, seguindo o padrão de modelo rico (rich domain model).
package entity
```

## 6. Testes

Criar testes unitários em `internal/helpers/*_test.go` para todas as funções puras:

- `moeda_test.go`: testar `FormatarMoeda` e `LimparNumero` com valores válidos e inválidos
- `string_test.go`: testar `Truncate` com strings curtas, longas e vazias
- `protocolo_test.go`: testar `GerarProtocolo` com range válido
- `data_test.go`: testar `ExtrairData` e `ExtrairHora` com timestamps completos e vazios
- `gerar_token_test.go`: testar `GerarToken` garantindo que retorna string não vazia

## 7. Critérios de Aceite

- [ ] `internal/entity/` compila sem erros
- [ ] `internal/helpers/` compila sem erros
- [ ] `go test ./internal/helpers/` passa
- [ ] Todas as funções possuem doc comment em português
- [ ] `doc.go` presente em ambos os pacotes
- [ ] Código antigo em `dominio/` e `processar.go` ainda compila (remoção será em SDD-026)
