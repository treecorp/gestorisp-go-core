# Decisao: Abordagem mista para queries (SELECT batch JOIN + UPDATE individual)

## Contexto
O `sync_conexoes_radius_status` fazia N consultas SELECT individuais a cada
iteracao do loop de contratos (1 por contrato). Com 600+ contratos, o tempo
de resposta ultrapassava 4 minutos devido a latencia de rede ate o MySQL.

## Decisao
Substituir N SELECTs individuais por 1 SELECT com JOIN, mantendo UPDATEs
individuais para preservar rastreabilidade:

```
ANTES:
  SELECT contratos (1 query)
  FOR each contrato:
    SELECT radacct WHERE acctuniqueid = ?  ← N queries (gargalo)
    if ativo: UPDATE contrato             ← 1 query

DEPOIS (misto):
  SELECT c.*, r.* FROM contratos c LEFT JOIN radacct r (1 query)
  FOR each linha:
    if ativo: UPDATE contrato             ← 1 query (igual)
```

## Onde se aplica
- `sync_conexoes_radius_status` → LEFT JOIN radacct no SELECT principal
- `reparar_offline_para_online` → INNER JOIN radacct no SELECT principal

## Onde NAO se aplica
- `sync_conexoes_radius` → ja e batch (1 query + N UPDATEs)
- `desbloquearUsuariosTravados` → RouterOS no meio do loop
- `reparar_online_para_offline` → ja e 1 query unica

## Consequencias
- SELECTs passam a ser <1s em vez de minutos
- UPDATEs continuam individuais — logs por contrato preservados
- Mesma atomicidade (autocommit, UPDATE WHERE id = ?)
- Zero risco de corrupcao — o JOIN so le dados, nao altera
