# SDD-027 — Limpeza de Código Morto (pós-refatoração v2)

**Status:** Planejado
**Autor:** Dev Backend
**Prioridade:** Alta
**Dependências:** SDD-018 a SDD-026

## 1. Objetivo

Remover tipos exportados não utilizados, funções nunca chamadas, duplicatas e campos de config mortos.

## 2. Escopo

### 2.1 Deletar (risco zero)

| # | Arquivo | O que |
|---|---------|-------|
| 1 | `internal/repositorio/cluster_repo.go` | ClusterContrato, CoordenadaGPS, LinhaConexao |
| 2 | `internal/repositorio/radacct_repo.go` | AtualizarRadacct (nunca chamada) |
| 3 | `internal/helpers/data.go` | ExtrairData + ExtrairHora (duplicadas em repositorio) |
| 4 | `internal/helpers/string.go` | Truncate (so no teste) |
| 5 | `internal/helpers/string_test.go` | Teste de Truncate |
| 6 | `internal/service/pagamento/baixa.go` | externalRef |
| 7 | `internal/service/bloqueio/cliente.go` | CalcularDiasAtraso (usar Fatura.CalcularDiasAtraso) |
| 8 | `internal/handler/gateway/server.go` | PingHandler |
| 9 | `internal/handler/gateway/webhook.go` | truncate |
| 10 | `internal/infra/banco/gispinstancia.go` | BuscarPopsOperacionais, AtualizarStatusTimeout, PingInstancia |
| 11 | `internal/config/config.go` | CI3EncryptionKey |

### 2.2 Refatorar

| # | O que | Acao |
|---|-------|------|
| 12 | responderJSON duplicado | Criar internal/helpers/http.go com tipo RespostaJSON + ResponderJSON |
| 13 | ExtrairData/ExtrairHora | Manter so em helpers/ (fonte unica). Atualizar cron_1.go |

## 3. Arquivos afetados

- internal/repositorio/cluster_repo.go — remover tipos
- internal/repositorio/radacct_repo.go — remover AtualizarRadacct, ExtrairData, ExtrairHora
- internal/helpers/data.go — manter (fonte unica)
- internal/helpers/string.go — remover Truncate + string_test.go
- internal/service/pagamento/baixa.go — remover externalRef
- internal/service/bloqueio/cliente.go — remover CalcularDiasAtraso
- internal/handler/gateway/server.go — remover PingHandler
- internal/handler/gateway/webhook.go — remover truncate
- internal/infra/banco/gispinstancia.go — remover 3 funcs
- internal/config/config.go — remover CI3EncryptionKey
- internal/helpers/http.go — NOVO (responderJSON unificado)
- internal/handler/worker/cron_1.go — atualizar import de repositorio → helpers
- internal/handler/gateway/server.go — importar responderJSON de helpers
- internal/handler/api/errors.go — importar responderJSON de helpers

## 4. Verificacao

go build ./... && go vet ./... && go test ./...
