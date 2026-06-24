# SDD-013 — Dashboard de Logs ao Vivo

**Status:** Aprovado
**Autor:** Dev Backend
**Prioridade:** Alta

## 1. Objetivo

Criar um painel web em tempo real para visualizar e analisar os logs gerados
pelos servicos gestor (cron) e worker, com filtros por instancia, tarefa e
nivel, alem de graficos de metrica de tempo de execucao.

## 2. Arquitetura

```
┌────────────────┐   HTTP POST (log events)   ┌──────────────────────┐
│  gestor (cron) │ ──────────────────────────> │    Dashboard         │
│  worker        │   /api/ingest               │  (Go net/http)      │
└────────────────┘                             │                      │
                                               │  Ring buffer 10.000  │
                                               │  Agreg. metricas     │
                                               │                      │
                                               │  /        → index.html│
                                               │  /api/ingest → POST  │
                                               │  /api/events → SSE   │
                                               │  /api/metricas → GET │
                                               └──────────┬───────────┘
                                                          │ SSE
                                               ┌──────────▼───────────┐
                                               │      Browser          │
                                               │  Tailwind CSS (CDN)   │
                                               │  Chart.js (CDN)      │
                                               └──────────────────────┘
```

## 3. Transporte em tempo real

**SSE (Server-Sent Events)** — sem dependencias externas, apenas `net/http`.

| Aspecto | Decisao |
|---------|---------|
| Transporte | SSE (`text/event-stream`) — zero novas deps Go |
| Reconexao | Nativa do browser (`EventSource`), reconecta automaticamente |
| Bidirecional | Nao necessario — filtros sao client-side |

## 4. Repositorio

**Mesmo repositorio** (`treecorp/gestorisp-go-core`), novo binario separado:

```
cmd/dashboard/main.go              ← Entry point do dashboard
internal/infra/observabilidade/    ← Pacote compartilhado
  hub.go                           ← SSE hub (broadcast para clientes)
  ingest.go                        ← Handlers HTTP de ingestao
  metricas.go                      ← Agregacao de metricas de tempo
web/dashboard/
  index.html                       ← UI unica (Tailwind + Chart.js via CDN)
```

## 5. Payload de log

```json
{
  "timestamp":   "2026/06/24 15:04:05",
  "nivel":       "info",
  "tag":         "cron_1",
  "instancia":   42,
  "mensagem":    "Instancia 42 processada com sucesso",
  "duracao_ms":  1250,
  "servico":     "worker"
}
```

## 6. Endpoints da API

| Metodo | Rota | Descricao |
|--------|------|-----------|
| `GET` | `/` | Pagina HTML do dashboard |
| `POST` | `/api/ingest` | Recebe logs dos servicos (worker/cron) |
| `GET` | `/api/events` | SSE stream de logs em tempo real |
| `GET` | `/api/metricas` | JSON com metricas agregadas (para chart inicial) |

## 7. Componentes da UI

### 7.1 Top bar
- Titulo "Dashboard Gestor ISP"
- Indicador de conexao SSE (verde/vermelho)

### 7.2 Barra de filtros (client-side)
- **Instancia:** `<select>` com "Todas" + lista de IDs distinct
- **Tarefa:** `<select>` com "Todas" + tags (cron_1, run_cluster, etc.)
- **Nivel:** `<select>` com "Todos" + info/sucesso/aviso/erro
- **Busca:** `<input>` com debounce para texto livre
- **Auto-scroll:** toggle on/off

### 7.3 Painel de logs (tabela)
- Colunas: horario, nivel (colorido), tag, instancia, mensagem
- Mais recentes no topo (ordem reversa)
- Scroll infinito com fallback para ring buffer
- Clique na linha expande JSON completo do payload

### 7.4 Painel de metricas
- **Card:** "Tempo medio de execucao" (ultimos N registros)
- **Card:** "Total execucoes (ultima hora)"
- **Card:** "Erros (ultima hora)" — vermelho
- **Grafico de barras:** "Tempo medio por handler" (Chart.js)
- **Grafico de pizza:** "Distribuicao por nivel" (Chart.js)
- **Grafico de linhas:** "Tempo de execucao ao longo do tempo" (Chart.js)

## 8. Alteracoes nos servicos existentes

### 8.1 Logger (`internal/infra/logger/logger.go`)
- Adicionar sistema de hooks: `AdicionarHook(fn func(nivel, tag, msg string))`
- Cada funcao de log executa os hooks apos imprimir

### 8.2 Config (`internal/config/config.go`)
- Adicionar `DashboardPort string` — lida de `DASHBOARD_PORT` (default `8080`)
- Adicionar `DashboardIngestURL string` — lida de `DASHBOARD_INGEST_URL`

### 8.3 Worker (`internal/worker/worker.go`)
- Envolver handler com cronometragem → preencher `duracao_ms` no log de conclusao

### 8.4 Cron (`internal/cron/tarefas/base.go`)
- Cronometrar execucao por instancia → incluir `duracao_ms` nos logs

### 8.5 Entry points (`cmd/gestor/main.go`, `cmd/worker/main.go`)
- Se `DashboardIngestURL` estiver configurado, registrar hook no logger
- O hook envia logs para o dashboard via HTTP POST (goroutine nao bloqueante)

### 8.6 Dockerfile
- Adicionar stage para compilar `cmd/dashboard`
- Copiar binario `dashboard` + diretorio `web/` para imagem final

### 8.7 CI/CD (`.github/workflows/ci.yml`)
- `go vet ./cmd/dashboard`

## 9. Configuracoes (env vars)

| Variavel | Default | Descricao |
|----------|---------|-----------|
| `DASHBOARD_PORT` | `8080` | Porta HTTP do servidor dashboard |
| `DASHBOARD_INGEST_URL` | `http://localhost:8080` | URL para worker/cron enviarem logs (vazio = desligado) |

## 10. Memorias (ring buffer)

Tipo: slice circular com capacidade fixa (10.000 entradas) para metricas.
SSE envia apenas entradas novas a partir do momento da conexao.

## 11. Nao faz parte desta SDD

- Autenticacao (painel interno, sem auth)
- Persistencia em banco (apenas memoria volatil)
- Deploy Kubernetes (sera tratado separadamente)
- Testes E2E do dashboard (apenas `go vet` + build)

## 12. Build

```bash
go vet ./cmd/dashboard && go build ./cmd/dashboard
go vet ./cmd/gestor ./cmd/worker && go build ./...
```
