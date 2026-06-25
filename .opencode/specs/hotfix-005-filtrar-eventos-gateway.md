# HOTFIX-005 — Filtrar eventos Iugu no gateway (apenas invoice.status_changed)

**Status:** Implementado
**Autor:** Dev Backend
**Prioridade:** Alta
**Tipo:** Correcao

## 1. Problema

O Iugu envia multiplos eventos para o mesmo webhook durante o ciclo de vida de uma fatura:

| Evento | Momento | Status |
|--------|---------|--------|
| `invoice.created` | Criacao da fatura | `pending` |
| `invoice.status_changed` | Mudanca de status (ex: paga) | `paid` |
| `invoice.released` | Liberacao do pagamento | `paid` |

O PHP legado so processava `invoice.status_changed`. O gateway Go atual publica **todos os eventos** na fila `processar_pagamento_iugu`, causando:

1. **Linhas orfas**: eventos como `invoice.created` e `invoice.released` sao inseridos em `gisp_iugu_gatilhos` mas nunca processados (`gisp_exec` fica `0`) porque as queries de UPDATE tem filtro `AND event = 'invoice.status_changed'`

2. **Ambiguidade no SELECT**: `SELECT gisp_exec FROM gisp_iugu_gatilhos WHERE id = ?` (sem filtro de event) retorna linha **arbitraria** quando ha multiplas para o mesmo `iuguFaturaID`. Se o MySQL retornar a linha `invoice.released` com `gisp_exec=0` em vez de `invoice.status_changed` com `gisp_exec=1`, o worker pode **tentar processar de novo** uma fatura ja baixada.

### 1.1 Sintomas observados em producao

```
ID=7C4CF4B1... Exec=0 Status=             Event=invoice.released   ⚠️ nao processado
ID=7C4CF4B1... Exec=1 Status=100001        Event=invoice.status_changed  ✅ processado
ID=7C4CF4B1... Exec=0 Status=             Event=invoice.created    ⚠️ nao processado
```

## 2. Correcao

### 2.1 Gateway — filtrar eventos na entrada

Em `internal/gateway/iugu_webhook.go`, apos extrair o `event` e validar que nao esta vazio, adicionar filtro:

```go
// Apenas invoice.status_changed eh processado (comportamento legado PHP)
if event != "invoice.status_changed" {
    logger.Info(tag, "Instancia %d: evento %s ignorado (apenas invoice.status_changed)", instancia.ID, event)
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("200"))
    return
}
```

Isso garante que **nenhum outro evento** chegue ao worker ou ao banco.

### 2.2 ProcessarPagamento — SELECT com filtro de event

Em `internal/pagamento/processar.go`, adicionar `AND event = 'invoice.status_changed'` no SELECT que verifica `gisp_exec`:

```go
// ANTES:
err := db.QueryRow(`SELECT COALESCE(gisp_exec, '0') FROM gisp_iugu_gatilhos WHERE id = ?`, iuguFaturaID).Scan(&gispExec)

// DEPOIS:
err := db.QueryRow(`SELECT COALESCE(gisp_exec, '0') FROM gisp_iugu_gatilhos WHERE id = ? AND event = 'invoice.status_changed'`, iuguFaturaID).Scan(&gispExec)
```

Isso elimina a ambiguidade: mesmo que por algum motivo outro evento chegue ao banco, o SELECT so considera linhas `invoice.status_changed`.

## 3. Impacto

- **Eventos ignorados**: `invoice.created`, `invoice.released` e qualquer outro evento sao ignorados silenciosamente (HTTP 200)
- **Zero risco** de processamento duplicado ou falho
- **Comportamento identico ao PHP legado**
- O webhook HTTP continua aceitando todos os Content-Types (form, JSON)
- Nao altera a API publica do gateway

## 4. Arquivos alterados

| Arquivo | Alteracao |
|---------|-----------|
| `internal/gateway/iugu_webhook.go` | Filtro de evento no inicio do handler |
| `internal/pagamento/processar.go` | `AND event = 'invoice.status_changed'` no SELECT |

## 5. Teste

- `go vet ./...` — sem erros
- `go build ./...` — compilacao limpa
- Enviar webhooks com eventos diferentes (`invoice.created`, `invoice.released`) — gateway retorna 200 sem processar
- Enviar `invoice.status_changed` — gateway publica na fila normalmente
