# HOTFIX-007 — Corrigir JOIN `buscarContrato` de `cliente_id` para `cliente_token`

**Status:** Implementado
**Autor:** Dev Backend
**Prioridade:** Alta
**Tipo:** Correcao

## 1. Problema

A funcao `buscarContrato` em `internal/pagamento/processar.go` faz o JOIN entre `sgp_clientes_contratos` e `sgp_clientes_new` usando `cliente_id`, mas o relacionamento correto no banco e por `cliente_token`.

### 1.1 Query atual (errada)

```go
SELECT c.id, c.token, c.status, c.cliente_id, 
    COALESCE(cli.pf_nome, cli.pj_razao_social, 'N/D') AS cliente_nome,
    c.cliente_token, c.pop_id, c.pppoe_user 
FROM sgp_clientes_contratos c
LEFT JOIN sgp_clientes_new cli ON cli.id = c.cliente_id   ← ERRADO
WHERE c.id = ?
```

### 1.2 Dois erros

**Erro 1 — JOIN pela coluna errada:**

`cli.id = c.cliente_id` mas `cliente_id` no contrato e geralmente `0` ou `NULL`. O relacionamento real e `cli.token = c.cliente_token`, conforme usado em TODO o codigo PHP legado:

```php
INNER JOIN sgp_clientes_new AS cliente ON cliente.token = contrato.cliente_token
```

**Erro 2 — NULL em campo `int`:**

`c.cliente_id` e `NULL` (ou `0`). Quando e `NULL` e o Go tenta escanear para `&c.ClienteID (int)`, o driver MySQL lanca erro:

```
sql: Scan error on column index 3, name "cliente_id": converting NULL to string is unsupported
```

### 1.3 Sintomas

- `buscarContrato` falha silenciosamente (log "nao encontrado")
- `criarProtocoloBaixa` nao insere registro em `sgp_clientes_contratos_protocolos`
- `desbloquearContratoDB` nao desbloqueia o contrato
- Protocolo **nunca e gerado** — mesmo com fatura paga e gatilho processado
- Bug existente desde a criacao do projeto (afeta todos os 7 handlers que usam `buscarContrato`)

## 2. Correcao

### 2.1 `internal/pagamento/processar.go` — query `buscarContrato`

```go
// ANTES:
err := q.QueryRow(`SELECT c.id, c.token, c.status, c.cliente_id, 
    COALESCE(cli.pf_nome, cli.pj_razao_social, 'N/D') AS cliente_nome,
    c.cliente_token, c.pop_id, c.pppoe_user 
    FROM sgp_clientes_contratos c
    LEFT JOIN sgp_clientes_new cli ON cli.id = c.cliente_id
    WHERE c.id = ?`, contratoID).Scan(...)

// DEPOIS:
err := q.QueryRow(`SELECT c.id, c.token, c.status, COALESCE(c.cliente_id, 0),
    COALESCE(cli.pf_nome, cli.pj_razao_social, 'N/D') AS cliente_nome,
    c.cliente_token, c.pop_id, c.pppoe_user 
    FROM sgp_clientes_contratos c
    LEFT JOIN sgp_clientes_new cli ON cli.token = c.cliente_token
    WHERE c.id = ?`, contratoID).Scan(...)
```

### 2.2 Mudancas

| Local | Antes | Depois |
|-------|-------|--------|
| SELECT `cliente_id` | `c.cliente_id` | `COALESCE(c.cliente_id, 0)` |
| JOIN condition | `cli.id = c.cliente_id` | `cli.token = c.cliente_token` |

Nao muda o numero de colunas nem a ordem — o Scan continua igual.

## 3. Impacto

- **Corrige** a geracao do protocolo de baixa em `sgp_clientes_contratos_protocolos`
- **Corrige** o desbloqueio do contrato apos pagamento
- **Nao quebra** nada — o `COALESCE` mantem compatibilidade com contratos que tem `cliente_id` valido
- `LEFT JOIN` mantido (contrato sem cliente em `sgp_clientes_new` ainda e retornado com nome 'N/D')
- Alinha com o comportamento do PHP legado

## 4. Arquivos alterados

| Arquivo | Alteracao |
|---------|-----------|
| `internal/pagamento/processar.go` | JOIN + COALESCE na query `buscarContrato` |

## 5. Teste

- `go vet ./...` — sem erros
- `go build ./...` — compilacao limpa
- Apos deploy, nova baixa deve gerar protocolo em `sgp_clientes_contratos_protocolos`
