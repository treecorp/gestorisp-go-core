# Gotcha: contrato_pop_id pode ser NULL na radacct

## Contexto
O `desbloquearUsuariosTravados` busca sessoes radacct e usa `contrato_pop_id`
para identificar qual POP esta associado a sessao.

## Sintoma
```
sql: Scan error on column index 3, name "contrato_pop_id":
converting NULL to int is unsupported
```

## Causa
A coluna `contrato_pop_id` na tabela `radacct` aceita NULL. O struct Go
usava `int` puro, que nao aceita NULL.

## CorrecAo
Usar `sql.NullInt64` no struct e validar com `.Valid` antes de usar:

```go
type sessaoTravada struct {
    AcctUpdateTime string
    RadAcctID      int
    AcctUniqueID   string
    ContratoPopID  sql.NullInt64   // ← nullable
    Username       string
}

// Uso:
if !s.ContratoPopID.Valid {
    logger.Info(... "sem POP vinculado, preservando")
    continue
}
pop, ok := pops[int(s.ContratoPopID.Int64)]
```

## Referencia
- `internal/worker/cron_1.go` — struct `sessaoTravada` linha 48

## Afeta
- Worker `cron_1` — `desbloquearUsuariosTravados`
- Instancias com sessoes radacct sem POP vinculado
