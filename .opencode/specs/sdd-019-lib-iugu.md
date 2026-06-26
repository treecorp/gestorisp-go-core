# SDD-019 — Lib Iugu

**Status:** Pendente
**Autor:** Knowledge Engineer
**Prioridade:** Alta
**Dependencias:** Nenhuma (cópia de `internal/pagamento/cliente_iugu.go`)

## 1. Objetivo

Criar `internal/lib/iugu/cliente.go` movendo a lógica de cliente HTTP Iugu de `internal/pagamento/cliente_iugu.go`, tornando-a reutilizável por services e handlers sem depender do pacote `pagamento`.

O pacote `internal/pagamento/cliente_iugu.go` deve permanecer **intacto** para não quebrar o código legado durante a migração.

## 2. Escopo

### 2.1 internal/lib/iugu/cliente.go

```go
// ClienteIugu é o cliente HTTP para a API de cobrança Iugu.
type ClienteIugu struct {
    apiURL string
    apiKey string
    http   *http.Client
}

// NovoClienteIugu cria uma nova instância de ClienteIugu.
func NovoClienteIugu(apiURL, apiKey string) *ClienteIugu

// ConsultarFatura consulta uma fatura na API Iugu.
func ConsultarFatura(cliente *ClienteIugu, faturaID string) (*FaturaIugu, error)
```

### 2.2 Structs de dados

```go
// FaturaIugu representa a resposta da API Iugu para uma fatura.
type FaturaIugu struct {
    ID           string `json:"id"`
    Status       string `json:"status"`
    Valor        string `json:"valor"`
    ValorPago    string `json:"valor_pago"`
    DataVencimento string `json:"data_vencimento"`
    DataPagamento string `json:"data_pagamento"`
    // demais campos conforme retorno da API
}
```

### 2.3 Manter original intacto

O arquivo `internal/pagamento/cliente_iugu.go` **não deve ser alterado**. Futuramente (SDD-026) será removido quando todos os imports forem atualizados.

## 3. Dependências

- `net/http` — chamadas HTTP
- `encoding/json` — parse de resposta
- `fmt` — formatação de erros
- `time` — timeout de requisição

## 4. Arquivos a criar

| Arquivo | Ação |
|---------|------|
| `internal/lib/iugu/cliente.go` | Criar — structs + funções |
| `internal/lib/iugu/doc.go` | Criar — package-level doc |

## 5. Documentação

Toda função e tipo deve ter **doc comment em português**:

```go
// Pacote iugu fornece um cliente HTTP para integração com a API de
// cobrança Iugu, permitindo consultar faturas e processar webhooks.
package iugu
```

Cada função exportada:

```go
// NovoClienteIugu cria e retorna um ClienteIugu configurado com a
// URL base e chave de API fornecidas.
func NovoClienteIugu(apiURL, apiKey string) *ClienteIugu

// ConsultarFatura busca os dados de uma fatura na API Iugu pelo seu
// identificador. Retorna erro se a fatura não existir ou a API falhar.
func ConsultarFatura(cliente *ClienteIugu, faturaID string) (*FaturaIugu, error)
```

## 6. Comportamento esperado

- `NovoClienteIugu` deve configurar timeout de 30s no `http.Client`
- `ConsultarFatura` deve fazer GET em `{apiURL}/faturas/{faturaID}` com header `Authorization: Basic {apiKey}`
- Em caso de erro HTTP (status != 200), retornar erro descritivo
- Em caso de timeout, retornar erro com contexto "iugu: timeout"

## 7. Critérios de Aceite

- [ ] `internal/lib/iugu/` compila sem erros
- [ ] `internal/pagamento/cliente_iugu.go` permanece intacto
- [ ] Todas as funções e tipos possuem doc comment em português
- [ ] `doc.go` presente no pacote
