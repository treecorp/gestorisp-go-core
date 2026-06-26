# SDD-024 — Handler Gateway + API

**Status:** Pendente
**Autor:** Knowledge Engineer
**Prioridade:** Alta
**Dependencias:** SDD-018 (`entity.Instancia`), SDD-019 (`lib/iugu`), SDD-022 (`service/pagamento`)

## 1. Objetivo

Mover `internal/gateway/` e `internal/api/` para `internal/handler/`, unificando a camada de transporte HTTP e simplificando os handlers para delegar lógica de negócio aos services. Centralizar `responderJSON` eliminando duplicação.

## 2. Escopo

### 2.1 internal/handler/gateway/server.go

```go
// Server configura e inicia o servidor HTTP do gateway na porta 8082.
func Server(port string, autenticar AutenticarFn, webhook WebhookFn) *http.Server
```

Setup de rotas do gateway legado, agora delegando para services.

### 2.2 internal/handler/gateway/auth.go

```go
// Autenticar valida o token de instância contra o banco de dados.
// Mantém acesso a DB pois é uma operação de infraestrutura.
func Autenticar(token string, db *sql.DB) (*entity.Instancia, error)
```

Extraído de `gateway/server.go`. Permanece usando `*sql.DB` por ser uma função de autenticação.

### 2.3 internal/handler/gateway/webhook.go

```go
// HandleWebhook processa requisições de webhook Iugu. Faz parse do
// request, valida dados e delega para o service de pagamento.
func HandleWebhook(w http.ResponseWriter, r *http.Request, instancia *entity.Instancia, rabbit *rabbit.Publisher)
```

Apenas parse HTTP e delegação — sem lógica de negócio.

### 2.4 internal/handler/api/server.go

```go
// Server configura e inicia o servidor HTTP da API na porta 8083.
func Server(port string, handlers ...Handler) *http.Server
```

Setup de rotas da API unificada (desconexão, gateway, swagger).

### 2.5 internal/handler/api/webhook_iugu.go

```go
// handlePagamentoIugu processa webhooks de pagamento Iugu na rota
// /api/v2/gateway/pagamentos/iugu/gatilho/{token}.
func handlePagamentoIugu(w http.ResponseWriter, r *http.Request)
```

Extraído de `api/server.go` para arquivo dedicado.

### 2.6 internal/handler/api/routeros.go

```go
// handleDesconectarPPPoE processa requisições de desconexão PPPoE na
// rota /api/v2/routeros/desconectarpppoe.
func handleDesconectarPPPoE(w http.ResponseWriter, r *http.Request, rabbit *rabbit.Publisher)
```

### 2.7 internal/handler/api/swagger.go

```go
// handleOpenAPI serve a spec OpenAPI embutida no binário.
func handleOpenAPI(w http.ResponseWriter, r *http.Request)

// handleSwaggerUI serve a interface Swagger UI via CDN.
func handleSwaggerUI(w http.ResponseWriter, r *http.Request)
```

### 2.8 internal/handler/api/errors.go

```go
// responderJSON serializa dados como JSON e escreve na resposta.
// Centralizado para evitar duplicação entre gateway e API.
func responderJSON(w http.ResponseWriter, status int, data interface{})

// handle404 retorna erro 404 padrão em JSON.
func handle404(w http.ResponseWriter, r *http.Request)
```

## 3. Mudanças específicas

- `responderJSON` é **centralizado** em `internal/handler/api/errors.go` (antes duplicado em `api/routeros_handler.go` e `gateway/server.go`)
- Gateway legado deve importar `handler/api/errors.go` ou melhor: mover `responderJSON` para um pacote compartilhado `internal/handler/response.go`
- Handlers não contêm lógica de negócio — apenas parse de request, validação básica, delegação para service e formatação de resposta

## 4. Arquivos a criar

| Arquivo | Ação |
|---------|------|
| `internal/handler/gateway/server.go` | Criar — setup HTTP porta 8082 |
| `internal/handler/gateway/auth.go` | Criar — Autenticar |
| `internal/handler/gateway/webhook.go` | Criar — HandleWebhook |
| `internal/handler/gateway/doc.go` | Criar — package-level doc |
| `internal/handler/api/server.go` | Criar — setup HTTP porta 8083 |
| `internal/handler/api/webhook_iugu.go` | Criar — handler webhook |
| `internal/handler/api/routeros.go` | Criar — handler desconexão |
| `internal/handler/api/swagger.go` | Criar — handlers OpenAPI + Swagger |
| `internal/handler/api/errors.go` | Criar — responderJSON + 404 |
| `internal/handler/api/doc.go` | Criar — package-level doc |
| `internal/handler/response.go` | Criar — responderJSON centralizado (se optar por separado) |

## 5. Documentação

Toda função deve ter **doc comment em português**:

```go
// Package gateway implementa os handlers HTTP para o gateway de
// pagamentos Iugu, responsável por receber webhooks e autenticar
// instâncias.
package gateway
```

```go
// responderJSON serializa o payload como JSON e escreve no
// http.ResponseWriter com o status HTTP informado. Centraliza a
// formatação de respostas JSON em todo o sistema.
func responderJSON(w http.ResponseWriter, status int, data interface{})
```

## 6. Critérios de Aceite

- [ ] `internal/handler/gateway/` compila sem erros
- [ ] `internal/handler/api/` compila sem erros
- [ ] `responderJSON` não está duplicado (única definição)
- [ ] Handlers não contêm SQL ou lógica de negócio (delegam para services)
- [ ] `cmd/gateway/main.go` e `cmd/api/main.go` atualizados para importar `handler/gateway` e `handler/api`
- [ ] Todas as funções possuem doc comment em português
