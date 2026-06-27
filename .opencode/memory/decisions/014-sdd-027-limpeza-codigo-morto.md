# Decisao 014 — SDD-027: Limpeza de Codigo Morto (pos-refatoracao v2)

**Data:** 26/06/2026
**Tipo:** Refatoracao / Cleanup
**Spec:** SDD-027

## Contexto

Apos a refatoracao estrutural v2 (SDD-018 a SDD-026), diversos tipos exportados,
funcoes e campos ficaram sem uso real (codigo morto). Tambem havia duplicacao de
`responderJSON`/`respostaJSON` entre pacotes `gateway` e `api`, e de
`ExtrairData`/`ExtrairHora` entre `repositorio` e `helpers`.

## Decisao

1. **Remocao risco zero:** Tipos/funcoes nunca chamados foram deletados.
2. **Unificacao de responderJSON:** Criado `internal/helpers/http.go` com
   `helpers.RespostaJSON` + `helpers.ResponderJSON`. Todos os handers (gateway e api)
   agora importam essa fonte unica.
3. **Fonte unica ExtrairData/ExtrairHora:** Mantido em `helpers/`. `cron_1.go` atualizado
   para chamar `helpers.ExtrairData`/`ExtrairHora`.

## Arquivos afetados

- `internal/repositorio/cluster_repo.go` — removidos ClusterContrato, CoordenadaGPS, LinhaConexao
- `internal/repositorio/radacct_repo.go` — removidos ExtrairData, ExtrairHora, AtualizarRadacct
- `internal/helpers/string.go` — removido Truncate
- `internal/helpers/string_test.go` — deletado
- `internal/service/pagamento/baixa.go` — removido externalRef
- `internal/service/bloqueio/cliente.go` — removido CalcularDiasAtraso
- `internal/handler/gateway/server.go` — removido PingHandler, local responderJSON/respostaJSON
- `internal/handler/gateway/webhook.go` — removido truncate
- `internal/infra/banco/gispinstancia.go` — removidos BuscarPopsOperacionais, AtualizarStatusTimeout, PingInstancia
- `internal/config/config.go` — removido CI3EncryptionKey
- `internal/helpers/http.go` — NOVO (RespostaJSON + ResponderJSON)
- `internal/handler/api/errors.go` — esvaziado (era local resposta/responderJSON)
- `internal/handler/api/routeros.go` — adaptado para helpers
- `internal/handler/api/server.go` — adaptado para helpers
- `internal/handler/api/swagger.go` — adaptado para helpers
- `internal/handler/api/webhook_iugu.go` — adaptado para helpers
- `internal/handler/worker/cron_1.go` — repositorio → helpers (ExtrairData/ExtrairHora)
