# Bug 002 — CAST dias_bloqueio retorna 0 para string vazia

**Status:** Corrigido (HOTFIX-004)

## Descricao

A coluna `dias_bloqueio` em `sgp_clientes_contratos` e `varchar(3) NULL`. O codigo Go usava `CAST(dias_bloqueio AS UNSIGNED)` no SQL para ler o valor. O `CAST` do MySQL retorna `0` para strings vazias ou nao numericas:

| Valor | `CAST(... AS UNSIGNED)` | Efeito |
|---|---|---|
| `NULL` | `NULL` | `Valid=false` → usa global ✅ |
| `''` | `0` | `Valid=true, Int64=0` — tolerancia zero 🔴 |
| `' '` | `0` | `Valid=true, Int64=0` — tolerancia zero 🔴 |
| `'abc'` | `0` | `Valid=true, Int64=0` — tolerancia zero 🔴 |

Isso causava bloqueio de contratos com 1 dia de atraso mesmo com `dias_bloqueio` global definido nos parametros.

## Correcao

Trocar `sql.NullInt64` + `CAST` para `sql.NullString` + parse manual com `strings.TrimSpace` + `strconv.Atoi`. String vazia, espacos ou caracteres nao numericos agora caem para o valor global.

## Arquivo afetado

`internal/worker/listar_clientes_vencidos.go`

## Spec

`specs/hotfix-004-cast-dias-bloqueio-varchar.md`
