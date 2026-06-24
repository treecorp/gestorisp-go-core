# SDD-009 — Handler limpeza_logs

**Status:** Aprovado  
**Autor:** Dev Backend  
**Prioridade:** Alta  
**Dependencias:** Nenhuma

## 1. Objetivo
Truncar (esvaziar) 5 tabelas de log/auditoria para evitar acumulo de disco.
Roda diariamente as 00:30.

## 2. Comportamento Esperado
- Conecta na instancia
- Para cada tabela na lista fixa, executa `TRUNCATE TABLE`
- Loga sucesso individual por tabela (ou erro, e continua)
- Se todas as 5 truncarem (ou falharem), retorna nil (nunca bloqueia o lote)

## 3. Tabelas
| # | Tabela | Conteudo |
|---|--------|----------|
| 1 | `sgp_clientes_logs` | Logs de atividade de clientes |
| 2 | `sgp_monitor_interfaces_historico` | Historico de monitoramento de interfaces |
| 3 | `sgp_webservices_cron` | Logs de execucao do cron |
| 4 | `radpostauth` | Logs de pos-autenticacao RADIUS |
| 5 | `SystemEvents` | Eventos do sistema (syslog) |

## 4. Contratos

### Handler
```go
func HandlerLimpezaLogs(instancia dominio.Instancia) error
```

### Fila RabbitMQ
`limpeza_logs` (ja declarada no cron em `cmd/gestor/main.go`)

### Cron
`0 30 0 * * *` — toda madrugada 00:30

## 5. SQL
```sql
TRUNCATE TABLE sgp_clientes_logs
TRUNCATE TABLE sgp_monitor_interfaces_historico
TRUNCATE TABLE sgp_webservices_cron
TRUNCATE TABLE radpostauth
TRUNCATE TABLE SystemEvents
```

## 6. Tratamento de Erros
- Falha de conexao → retorna erro → worker Nack
- Falha em uma tabela especifica → loga erro, `continue` para a proxima
- Todas as tabelas falharem → retorna nil (nao bloqueia mensagem, pois o
  sistema nao deve travar por tabela ausente)
- Sucesso em todas → retorna nil → worker Ack

## 7. Registro no Worker
Incluir no slice de `Consumidor` em `cmd/worker/main.go`:
```go
{
    Fila:    "limpeza_logs",
    Handler: worker.HandlerLimpezaLogs,
},
```

## 8. Estimativa
~45 linhas, 1 arquivo novo, 0 dependencias
