# SDD-017 — API REST (cmd/api) — Desconexao PPPoE + Gateway Iugu

**Status:** Implementado
**Autor:** Dev Backend
**Prioridade:** Media
**Dependencias:** Infra existente (`mensageria`, `dominio`, `fuso`, `logger`, `config`, `gateway`, `banco`)

## 1. Objetivo

Criar uma API REST independente (`cmd/api`, porta `8083`) que unifica:

1. **Desconexao PPPoE** — recebe dados do RouterOS e publica na fila `desconectar_contrato`
2. **Gateway Iugu** — recebe webhooks de pagamento na mesma rota do gateway legado (`/api/v2/gateway/pagamentos/iugu/gatilho/{token}`), reaproveitando as funcoes `Autenticar` + `HandleWebhook` do pacote `internal/gateway`
3. **JSON puro** — todas as respostas (inclusive erros 404, 405, 500) retornam JSON, nunca HTML
4. **OpenAPI + Swagger UI** — spec documentada em `/openapi.yaml` e UI via CDN em `/swagger`

Tambem corrige o gateway legado (`internal/gateway/`) para retornar JSON em vez de HTML em todos os `http.Error`.

## 2. Arquitetura

```
Porta 8083 (cmd/api)
├── POST /api/v2/routeros/desconectarpppoe
│     └── Valida JSON → Publica "desconectar_contrato" → 200 JSON
│
├── POST /api/v2/gateway/pagamentos/iugu/gatilho/{token}
│     └── Autenticar token → HandleWebhook (reaproveita gateway) → 200 JSON
│
├── GET /openapi.yaml
│     └── Spec OpenAPI 3.0.3 (embedded)
│
├── GET /swagger
│     └── Swagger UI via CDN (Tailwind-style)
│
└── /* (catch-all)
      └── 404 JSON
```

## 3. Endpoints

### 3.1 `POST /api/v2/routeros/desconectarpppoe`

Request:
```json
{
  "instancia_id": 1,
  "contrato_id": 1113,
  "cliente_nome": "Fulano",
  "pppoe_user": "fulano@isp",
  "pop_ipv4": "177.136.249.55",
  "pop_port": "8728",
  "pop_user": "admin",
  "pop_pass": "senha"
}
```

Response 200:
```json
{"sucesso": true, "mensagem": "Publicado na fila desconectar_contrato"}
```

Response 400:
```json
{"sucesso": false, "erro": "pppoe_user é obrigatorio"}
```

### 3.2 `POST /api/v2/gateway/pagamentos/iugu/gatilho/{token}`

Comportamento identico ao gateway legado (`POST /pagamentos/iugu/gatilho/{token}`),
reaproveitando `gateway.Autenticar` + `gateway.HandleWebhook`.

### 3.3 `GET /openapi.yaml`

Retorna a spec OpenAPI 3.0.3 embedded no binario.

### 3.4 `GET /swagger`

Pagina HTML com Swagger UI carregado via CDN, apontando para `/openapi.yaml`.

## 4. JSON em todas as respostas

### API nova (cmd/api)
- Handler `HandleDesconectarPPPoE` — ja usa `responderJSON` em todos os caminhos
- Catch-all `/*` — retorna `{"sucesso": false, "erro": "Rota nao encontrada"}` (HTTP 404)

### Gateway legado (cmd/gateway)
- `internal/gateway/server.go`: `http.Error("Token nao informado")` → JSON
- `internal/gateway/server.go`: `http.Error("Nao permitido")` → JSON
- `internal/gateway/iugu_webhook.go`: 5 `http.Error()` → JSON
- `w.Write([]byte("200"))` → `{"sucesso": true, "mensagem": "200"}`

## 5. Campos Obrigatorios (Desconexao PPPoE)

| Campo | Tipo | Obrigatorio | Descricao |
|-------|------|-------------|-----------|
| `pppoe_user` | string | Sim | Login PPPoE do cliente |
| `pop_ipv4` | string | Sim | IP do RouterOS |
| `pop_port` | string | Sim | Porta API RouterOS |
| `pop_user` | string | Sim | Usuario RouterOS |
| `pop_pass` | string | Sim | Senha RouterOS |
| `instancia_id` | int | Nao | ID da instancia GISP |
| `contrato_id` | int | Nao | ID do contrato |
| `cliente_nome` | string | Nao | Nome do cliente |

## 6. Binarios e Portas

| Binario | Porta | Uso |
|---------|-------|-----|
| `cmd/gateway` | 8082 | Gateway Iugu legado (inalterado) |
| `cmd/api` | 8083 | API unificada (desconexao + gateway + swagger) |

## 7. Configuracao

| Env | Default | Descricao |
|-----|---------|-----------|
| `API_PORT` | `8083` | Porta do servidor API |
| `GATEWAY_PORT` | `8082` | Porta do gateway legado (inalterado) |

## 8. Arquivos Envolvidos

| Arquivo | Acao |
|---------|------|
| `cmd/api/main.go` | Criado — entrypoint + conexao banco |
| `internal/api/server.go` | Criado — rotas, swagger, 404, handler gateway |
| `internal/api/routeros_handler.go` | Criado — handler desconexao PPPoE |
| `internal/api/openapi.yaml` | Criado — spec OpenAPI embedded |
| `internal/gateway/server.go` | Alterado — `http.Error` → `responderJSON` |
| `internal/gateway/iugu_webhook.go` | Alterado — `http.Error` → `responderJSON` |
| `internal/config/config.go` | Alterado — adicionado `APIPort` |
| `Dockerfile` | Alterado — build + copy + entrypoint `api` |
| `.opencode/memory/index.md` | Alterado — update stats + rotas |
