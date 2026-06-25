# HOTFIX-004 — Parse `dias_bloqueio` varchar com fallup seguro no bloqueio de inadimplentes

**Status:** Pendente
**Autor:** Dev Backend
**Prioridade:** Alta
**Tipo:** Correcao

## 1. Problema

A coluna `dias_bloqueio` em `sgp_clientes_contratos` e `sgp_parametros` e `varchar(3) NULL`. O codigo Go usa `CAST(dias_bloqueio AS UNSIGNED)` no SQL que, segundo comportamento do MySQL:

| Valor no banco | `CAST(.. AS UNSIGNED)` | Efeito no Go |
|---|---|---|
| `NULL` | `NULL` | `Valid=false` → usa global ✅ |
| `''` (vazio) | `0` | `Valid=true, Int64=0` — **tolerancia zero** 🔴 |
| `' '` (espacos) | `0` | `Valid=true, Int64=0` — **tolerancia zero** 🔴 |
| `'abc'` | `0` | `Valid=true, Int64=0` — **tolerancia zero** 🔴 |
| `'5'` | `5` | OK ✅ |
| `'0'` | `0` | `Valid=true, Int64=0` — bloqueio imediato (se foi intencional) ⚠️ |

O sistema e legado e pode conter dados sujos nessa coluna (espacos, string vazia, caracteres nao numericos). Quando isso acontece, o contrato fica com `diasBloqueio = 0`, ou seja, **bloqueio no 1o dia de atraso** — ignorando o valor global definido nos parametros.

### 1.1 Sintomas

- Contratos sendo bloqueados com 1 dia de atraso mesmo com `dias_bloqueio` global = 5
- Diferenca de comportamento entre PHP legado (que ignora a coluna contratual e so usa o global) e Go (que le a coluna e o CAST corrompe)
- Impussivel distinguir entre `dias_bloqueio=0` intencional (bloquear imediatamente) e `dias_bloqueio=''` (deveria ser NULL → usar global)

## 2. Correcao

Trocar o scan de `sql.NullInt64` com `CAST` para `sql.NullString` com parse manual em Go.

### 2.1 `internal/worker/listar_clientes_vencidos.go`

#### Struct `contratoBloqueio` (linha 33)

```go
// ANTES:
DiasBloqueio sql.NullInt64

// DEPOIS:
DiasBloqueio sql.NullString
```

#### Query `lerContrato` (linha 324)

```sql
-- ANTES:
CAST(dias_bloqueio AS UNSIGNED) AS dias_bloqueio

-- DEPOIS:
dias_bloqueio
```

#### Logica `processarFatura` (linhas 184-187)

```go
// ANTES:
diasBloqueio := diasBloqueioGlobal
if contrato.DiasBloqueio.Valid {
    diasBloqueio = int(contrato.DiasBloqueio.Int64)
}

// DEPOIS:
diasBloqueio := diasBloqueioGlobal
if contrato.DiasBloqueio.Valid {
    trimmed := strings.TrimSpace(contrato.DiasBloqueio.String)
    if trimmed != "" {
        if val, err := strconv.Atoi(trimmed); err == nil {
            diasBloqueio = val
        }
    }
}
```

#### Imports

Adicionar `strconv` aos imports.

### 2.2 Comportamento apos correcao

| Valor no banco | Antes (`CAST`) | Depois (parse Go) |
|---|---|---|
| `NULL` | `Valid=false` → global | `Valid=false` → global ✅ |
| `''` | `Valid=true, Int64=0` 🔴 | `Valid=true, Trim=""` → global ✅ |
| `' '` | `Valid=true, Int64=0` 🔴 | `Valid=true, Trim=""` → global ✅ |
| `'5'` | `Valid=true, Int64=5` | `Valid=true, Atoi=5` ✅ |
| `'abc'` | `Valid=true, Int64=0` 🔴 | `Valid=true, Atoi=erro` → global ✅ |
| `'0'` | `Valid=true, Int64=0` | `Valid=true, Atoi=0` → bloqueio imediato ✅ (igual, mas intencional) |
| `' 5 '` | `Valid=true, Int64=0` 🔴 | `Valid=true, Trim="5", Atoi=5` ✅ |

## 3. Impacto

- **Nenhuma quebra de comportamento** para dados limpos (NULL ou numeros validos)
- **Corrige** bloqueio prematuro de contratos com dados sujos na coluna
- Compatibilidade total com PHP legado (que ignora a coluna contratual)
- Per-contract `dias_bloqueio` continua funcional para quem tem valor numerico valido

## 4. Arquivos alterados

| Arquivo | Alteracao |
|---|---|
| `internal/worker/listar_clientes_vencidos.go` | Tipo do campo, query, logica de parse, import |

## 5. Teste

- `go vet ./...` — sem erros
- `go build ./...` — compilacao limpa
- Simular contratos com: `NULL`, `''`, `' '`, `'5'`, `'abc'`, `' 5 '`, `'0'` — todos devem se comportar como especificado na tabela 2.2
