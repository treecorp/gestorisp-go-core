# SDD-028 — Smoke Test: Validação de Todos os Binários cmd/ em Produção

**Status:** Planejado
**Autor:** Dev Backend
**Prioridade:** Crítica
**Dependências:** SDD-027 (limpeza concluída), build + vet + tests passando

---

## 1. Objetivo

Criar uma **spec de validação (smoke test)** que garanta que os **5 binários** (`gestor`, `worker`, `gateway`, `api`, `testedesconexao`) compilam, iniciam sem crash, conectam nas dependências (MySQL GISPADM, RabbitMQ) e expõem os endpoints esperados — **antes do deploy em produção**.

O dashboard (`gui-monitor`) está em branch separado e será validado separadamente quando mergeado.

---

## 2. Escopo

| Binário | Entry Point | Porta | Tipo | Dependências Externas |
|---------|-------------|-------|------|----------------------|
| `gestor` | `cmd/gestor/main.go` | — | Cron scheduler | MySQL GISPADM, RabbitMQ |
| `worker` | `cmd/worker/main.go` | — | Consumer RabbitMQ | MySQL GISPADM, RabbitMQ |
| `gateway` | `cmd/gateway/main.go` | 8082 | HTTP webhook Iugu | MySQL GISPADM, RabbitMQ |
| `api` | `cmd/api/main.go` | 8083 | HTTP REST + OpenAPI | MySQL GISPADM, RabbitMQ |
| `testedesconexao` | `cmd/testedesconexao/main.go` | — | CLI RouterOS | RouterOS (opcional) |

---

## 3. Cenários de Teste

### 3.1 Build + Vet + Testes Unitários (Gate obrigatório)

```bash
go build ./... && go vet ./... && go test ./internal/helpers/... ./internal/entity/...
```

**Critério:** Exit code 0, sem warnings de vet.

---

### 3.2 Binários Compilam Individualmente

```bash
go build -o /tmp/gestor   ./cmd/gestor
go build -o /tmp/worker   ./cmd/worker
go build -o /tmp/gateway  ./cmd/gateway
go build -o /tmp/api      ./cmd/api
go build -o /tmp/testedesconexao ./cmd/testedesconexao
```

**Critério:** Todos geram executável sem erro.

---

### 3.3 Gestor (Cron) — Inicialização + Conexões + Agendador

**Pré-condição:** MySQL GISPADM acessível, RabbitMQ acessível.

**Passos:**
1. `./gestor` (com env vars válidas)
2. Aguardar logs:
   - `gestor: Aguardando conexao com banco GISPADM...`
   - `gestor: Aguardando conexao com RabbitMQ...`
   - `gestor: Conexoes estabelecidas. Iniciando agendador...`
3. Verificar que 6 tarefas são registradas (exceto `listar_clientes_vencidos` comentada):
   - `cron_um` (expressão `0 */1 * * * *`, fila `cron_1`)
   - `executar_cluster` (fila `run_cluster`)
   - `verificar_status_pop` (fila `check_pop_status`)
   - `sincronizar_conexoes` (fila `sync_conexoes_radius_arquivo`)
   - `reparar_radius` (fila `repair_radius_acctstoptime`)
   - `limpeza_logs` (fila `limpeza_logs`)
4. Enviar `SIGINT` (Ctrl+C) → log `gestor: Sinal recebido: interrupt. Encerrando...` → `gestor: Gestor encerrado com sucesso`

**Critério:** Processo inicia, conecta, agenda, encerra graciosamente (exit 0).

---

### 3.4 Worker — Inicialização + Consumidores Registrados

**Pré-condição:** RabbitMQ acessível.

**Passos:**
1. `./worker`
2. Aguardar logs:
   - `worker: Aguardando conexao com RabbitMQ...`
   - `worker: Conectado ao RabbitMQ. Iniciando consumidores...`
3. Verificar 7 consumidores de fila (via `Iniciar`):
   - `check_pop_status` → `HandlerCheckPopStatus`
   - `run_cluster` → `HandlerRunCluster`
   - `sync_conexoes_radius_arquivo` → `HandlerSyncConexoesRadiusArquivo`
   - `cron_1` → `HandlerCron1`
   - `repair_radius_acctstoptime` → `HandlerRepairRadiusAcctstoptime`
   - `limpeza_logs` → `HandlerLimpezaLogs`
   - `listar_clientes_vencidos` → `HandlerListarClientesVencidos`
4. Verificar 2 consumidores de mensagem (via `IniciarMensagem`):
   - `processar_pagamento_iugu` → `HandlerProcessarPagamentoIugu` (retry infinito)
   - `desconectar_contrato` → `HandlerDesconectarContrato` (retry infinito)
5. `SIGINT` → encerramento gracioso.

---

### 3.5 Gateway (Porta 8082) — HTTP + Webhook Iugu

**Pré-condição:** MySQL GISPADM + RabbitMQ.

**Passos:**
1. `./gateway` (env `GATEWAY_PORT=8082`)
2. Aguardar:
   - `gateway: Conectando ao banco global GISPADM...`
   - `gateway: Conectado ao banco GISPADM`
   - `gateway: Conectando ao RabbitMQ...`
   - `gateway: Conectado ao RabbitMQ`
   - `gateway: Servidor HTTP ouvindo na porta 8082`
3. **Health check implícito:** `curl -s -o /dev/null -w "%{http_code}" http://localhost:8082/pagamentos/iugu/gatilho/TOKEN_INEXISTENTE` → **403** (token inválido) ou **400** (token vazio)
4. **Formato PHP (form-urlencoded):** `curl -X POST "http://localhost:8082/pagamentos/iugu/gatilho/TOKEN_VALIDO" -d "event=invoice.status_changed&data[id]=INV123&data[status]=paid&data[payer_name]=Teste"` → **200** + JSON `{"sucesso":true,"mensagem":"Publicado na fila processar_pagamento_iugu"}` (ou 200 silencioso se evento ignorado)
5. **JSON puro:** `curl -X POST "http://localhost:8082/pagamentos/iugu/gatilho/TOKEN_VALIDO" -H "Content-Type: application/json" -d '{"event":"invoice.status_changed","data":{"id":"INV123","status":"paid","payer_name":"Teste"}}'` → **200**
6. **Método errado:** `curl -X GET ...` → **405** + JSON erro
7. `SIGINT` → shutdown gracioso (timeout 10s).

---

### 3.6 API (Porta 8083) — REST + OpenAPI + Swagger

**Pré-condição:** MySQL GISPADM + RabbitMQ.

**Passos:**
1. `./api` (env `API_PORT=8083`)
2. Aguardar: `api: Servidor HTTP ouvindo na porta 8083`
3. **Rota 404:** `curl -s http://localhost:8083/rota-inexistente` → **404** + JSON `{"sucesso":false,"erro":"Rota nao encontrada"}`
4. **OpenAPI YAML:** `curl -s http://localhost:8083/openapi.yaml` → **200** + conteúdo YAML válido (contém `openapi: 3.0.3`, `paths:`, `components:`)
5. **Swagger UI:** `curl -s http://localhost:8083/swagger` → **200** + HTML (contém `swagger-ui`)
6. **POST /api/v2/routeros/desconectarpppoe:**
   - Payload válido (JSON com `pppoe_user`, `pop_ipv4`, `pop_port`, `pop_user`, `pop_pass`) → **200** + `{"sucesso":true,"mensagem":"Publicado na fila desconectar_contrato"}`
   - Payload faltando `pppoe_user` → **400** + JSON erro
   - Método GET → **405** + JSON erro
7. **POST /api/v2/gateway/pagamentos/iugu/gatilho/{token}:** Mesmo comportamento do gateway (403 token inválido, 200 webhook válido)
8. `SIGINT` → shutdown gracioso (timeout 10s).

---

### 3.7 Testedesconexao — CLI RouterOS (Modo Consulta)

**Pré-condição:** RouterOS acessível (IP/porta/credenciais hardcoded no código).

**Passos:**
1. `./testedesconexao` (sem flags)
2. Aguardar logs:
   - `testedesconexao: Conectando a 10.20.1.2:8728...`
   - `testedesconexao: Conectado ao RouterOS`
   - `testedesconexao: Usuario 04720186475 NAO esta ativo...` OU `testedesconexao: Usuario 04720186475 esta ATIVO...`
   - `testedesconexao: Modo consulta apenas. Use --executar para desconectar.`
3. Exit code 0.

**Nota:** Se RouterOS indisponível, falha esperada (exit 1) — documentar como "dependência externa".

---

### 3.8 Docker Build + Imagem (Validação CI/CD)

```bash
docker build -t gestorisp-smoke .
docker run --rm --entrypoint sh gestorisp-smoke -c "ls -la /gestor /worker /gateway /api /testedesconexao"
```

**Critério:** 5 binários presentes no `/app` da imagem final.

---

## 4. Variáveis de Ambiente Necessárias (para smoke test local)

```bash
# Banco GISPADM (produção)
export DB_GISPADM_HOST=177.136.249.51
export DB_GISPADM_PORT=31034
export DB_GISPADM_USER=gestorisp
export DB_GISPADM_PASS=WM33223200kl**
export DB_GISPADM_DBNAME=gisp_adm

# RabbitMQ
export RABBITMQ_HOST=172.16.12.10
export RABBITMQ_PORT=31837
export RABBITMQ_USER=guest
export RABBITMQ_PASS=guest

# Portas HTTP
export GATEWAY_PORT=8082
export API_PORT=8083
```

---

## 5. Checklist de Validação (Para o PR/Deploy)

| # | Item | OK? |
|---|------|-----|
| 1 | `go build ./...` passa | ☐ |
| 2 | `go vet ./...` passa | ☐ |
| 3 | `go test ./internal/helpers/... ./internal/entity/...` passa | ☐ |
| 4 | 5 binários compilam individualmente | ☐ |
| 5 | `gestor` inicia + conecta DB + Rabbit + agenda 6 tarefas + SIGINT OK | ☐ |
| 6 | `worker` inicia + conecta Rabbit + 7 consumidores fila + 2 consumidores mensagem + SIGINT OK | ☐ |
| 7 | `gateway` porta 8082 responde 403/400/200/405 corretamente | ☐ |
| 8 | `api` porta 8083: 404 JSON, OpenAPI YAML, Swagger UI, 2 rotas POST OK | ☐ |
| 9 | `testedesconexao` modo consulta executa sem crash | ☐ |
| 10 | `docker build` gera imagem com 5 binários | ☐ |
| 11 | CI/CD (GitHub Actions) roda build+vet+test nas branches `main` e `v2` | ☐ |

---

## 6. Arquivos Afetados / Relacionados

- `.github/workflows/ci.yml` — adicionar step de smoke test (opcional, pode ser script separado)
- `cmd/gestor/main.go`, `cmd/worker/main.go`, `cmd/gateway/main.go`, `cmd/api/main.go`, `cmd/testedesconexao/main.go`
- `internal/config/config.go` — env vars
- `Dockerfile` — build dos 5 binários
- `docs/arquitetura.md` — atualizar se necessário

---

## 7. Próximos Passos (Pós-Spec)

1. **Criar script automatizado** `scripts/smoke-test.sh` que roda todos cenários acima
2. **Integrar no CI** como job separado `smoke-test` (requer MySQL + RabbitMQ de teste — pode usar testcontainers ou services do GitHub Actions)
3. **Documentar no README** como rodar smoke test localmente
4. **Validar em staging** antes de promover para produção