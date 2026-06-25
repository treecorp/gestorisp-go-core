# Gotcha 008 — CAST('' AS UNSIGNED) retorna 0, nao NULL

**Data:** 25/06/2026
**Contexto:** HOTFIX-004, coluna `dias_bloqueio varchar(3) NULL` em `sgp_clientes_contratos`.

## Problema
O MySQL `CAST('' AS UNSIGNED)` retorna `0`, nao `NULL`. Isso fazia com que `sql.NullInt64.Valid = true` mesmo para dados sujos (string vazia, espacos, caracteres nao numericos), resultando em `diasBloqueio = 0` — bloqueio imediato no 1o dia de atraso.

## Licao
Nunca confiar em `CAST(coluna AS UNSIGNED)` do MySQL para colunas varchar que podem conter dados sujos. Preferir `sql.NullString` + parse manual em Go com `strings.TrimSpace` + `strconv.Atoi`.

## Resumo
| Valor no banco | `CAST(... AS UNSIGNED)` | Parse Go (`TrimSpace` + `Atoi`) |
|---|---|---|
| `NULL` | `NULL` | `Valid=false` → global |
| `''` | `0` 🔴 | `Trim=""` → global ✅ |
| `' '` | `0` 🔴 | `Trim=""` → global ✅ |
| `'abc'` | `0` 🔴 | `Atoi=erro` → global ✅ |
| `'5'` | `5` | `Atoi=5` ✅ |
