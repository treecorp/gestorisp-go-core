# Gotcha: Colunas IPv6 nao existem em radacct — SELECT fixo quebra query

## Contexto
O `sync_conexoes_radius_arquivo` faz SELECT da tabela `radacct` e INSERT em
`radacct_arquivo`. Colunas IPv6 (`framedipv6pool`, `framedipv6prefix`,
`delegatedipv6prefix`, `mikrotikrealm`) podem existir em `radacct_arquivo`
mas **nao** em `radacct`.

## Sintoma
Zero registros arquivados. A query SELECT falha silenciosamente com
"Unknown column 'framedipv6pool'" e o handler loga erro e Nack a mensagem.

## Causa
O PHP original usava `SELECT *`, que se adapta automaticamente ao schema.
A porta Go listava colunas fixas incluindo as 4 IPv6. Se `radacct` nao tem
essas colunas, o MySQL rejeita a query.

## CorrecAo
Detectar colunas de `radacct` via `INFORMATION_SCHEMA` (igual ja era feito
para `radacct_arquivo`) e montar SELECT + Scan dinamicamente:

```go
colunasSELECT := montarListaColunasSELECT(colunasRadacct)
query := fmt.Sprintf("SELECT %s FROM radacct WHERE ...", colunasSELECT)

// Scan tambem dinâmico:
targets := montarScanTargets(&r, colunasRadacct)
linhas.Scan(targets...)
```

Funcoes novas:
- `detectarColunasRadacct(db)` — detecta colunas em `radacct`
- `montarListaColunasSELECT(colunasRadacct)` — monta lista para SELECT
- `montarScanTargets(r, colunasRadacct)` — monta destinos para Scan

## Referencia
- `internal/worker/sync_conexoes_radius_arquivo.go` — `buscarRadacctPendenteArquivo` linha 183

## Afeta
- Worker `sync_conexoes_radius_arquivo`
- Instancias onde `radacct` foi criado sem colunas IPv6 (instancias antigas)
