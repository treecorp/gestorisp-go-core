# SDD-008 — Handler repair_radius_acctstoptime

**Status:** Aprovado  
**Autor:** Dev Backend  
**Prioridade:** Alta  
**Dependências:** Nenhuma

## 1. Objetivo
Remover registros orfaos da tabela `radacct` (sessoes RADIUS nunca finalizadas)
que possuem `acctstoptime IS NULL`. Roda diariamente as 00:30.

## 2. Comportamento Esperado
- Conecta na instancia, executa `DELETE FROM radacct WHERE acctstoptime IS NULL`
- Loga quantos registros foram removidos (ou "nenhum" se zero)
- Se a query falhar, retorna erro (worker Nack a mensagem)
- Se a query suceder, Ack a mensagem

## 3. Contratos

### Handler
```go
func HandlerRepairRadiusAcctstoptime(instancia dominio.Instancia) error
```

### Fila RabbitMQ
`repair_radius_acctstoptime` (ja declarada no cron em `cmd/gestor/main.go`)

### Cron
`0 30 0 * * *` — toda madrugada 00:30

## 4. SQL
```sql
DELETE FROM radacct WHERE acctstoptime IS NULL
```

## 5. Tratamento de Erros
- Falha de conexao → retorna erro (`fmt.Errorf`) → worker Nack
- Falha na query → retorna erro → worker Nack
- Sucesso → `RowsAffected` logado como info/sucesso → worker Ack

## 6. Registro no Worker
Incluir no slice de `Consumidor` em `cmd/worker/main.go`:
```go
{
    Fila:    "repair_radius_acctstoptime",
    Handler: worker.HandlerRepairRadiusAcctstoptime,
},
```

## 7. Estimativa
~35 linhas, 1 arquivo novo, 0 dependencias
