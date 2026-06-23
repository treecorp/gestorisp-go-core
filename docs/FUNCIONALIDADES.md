
# Funcionalidades - Tarefas Cron

## Visao Geral

O Gestor ISP possui **7 tarefas cron** agendadas que substituem as funcionalidades do antigo microsservico `gestorisp-ws-cron` em PHP.

Cada tarefa segue o mesmo padrao:

```
1. Conectar no GISPADM (MySQL central)
2. Buscar todas as instancias ativas (GISP-FULL)
3. Para cada instancia:
   a. Montar payload com credenciais da instancia
   b. Serializar como JSON
   c. Codificar em Base64
   d. Publicar na fila RabbitMQ correspondente
4. Logar resultados
```

## Tabela de Tarefas

| # | Tarefa | Agendamento | Fila RabbitMQ | Prioridade |
|---|---|---|---|---|
| 1 | `cron_um` | `*/1 * * * *` (a cada 1 min) | `cron_1` | Alta |
| 2 | `executar_cluster` | `*/6 0,3-23 * * *` (a cada 6 min) | `run_cluster` | Alta |
| 3 | `verificar_status_pop` | `* * * * *` (a cada 1 min) | `check_pop_status` | Alta |
| 4 | `sincronizar_conexoes` | `* * * * *` (a cada 1 min) | `sync_conexoes_radius_arquivo` | Alta |
| 5 | `reparar_radius` | `30 0 * * *` (diario 00:30) | `repair_radius_acctstoptime` | Media |
| 6 | `limpeza_logs` | `30 0 * * *` (diario 00:30) | `limpeza_logs` | Baixa |
| 7 | `listar_clientes_vencidos` | `10 14 * * *` (diario 14:10) | `listar_clientes_vencidos` | Alta |

## Detalhamento das Tarefas

### 1. cron_um

**Agendamento original:** `*/5 0,3-23 * * *` (a cada 5 min, exceto 1-2h)
**Agendamento atual (temporario):** `* * * * *` (a cada 1 min)
**Fila RabbitMQ:** `cron_1`

**Descricao:**
Tarefa geral de manutencao do sistema. No backend PHP, executa:

- `sync_1()` — sincronizacao geral de dados
- `sync_conexoes_radius_status_reparar()` — repara status de conexoes Radius
- `sync_conexoes_radius_status_reparar_online_to_offline()` — corrige status online para offline

**Dados trafegados:**
```json
{
  "gisp_id": 12,
  "gisp_token": "abc123",
  "hostname": "177.136.249.51",
  "username": "root",
  "password": "senha",
  "database": "gisp_cliente"
}
```

### 2. executar_cluster

**Agendamento:** `*/6 0,3-23 * * *` (a cada 6 min, exceto 1-2h)
**Fila RabbitMQ:** `run_cluster`

**Descricao:**
Atualiza o mapa de cluster de contratos. No backend PHP:

- `run_cluster_contratos()` — atualiza agrupamento de contratos
- `run_contratos_coordenadas()` — atualiza coordenadas geograficas

### 3. verificar_status_pop

**Agendamento:** `* * * * *` (a cada 1 min)
**Fila RabbitMQ:** `check_pop_status`

**Descricao:**
Verifica o status operacional de todos os POPs (Pontos de Presenca) cadastrados.
No backend PHP, para cada POP:

- Carrega dados do POP via RouterOS API
- Verifica se esta OPERACIONAL
- Atualiza status no banco

### 4. sincronizar_conexoes

**Agendamento:** `* * * * *` (a cada 1 min)
**Fila RabbitMQ:** `sync_conexoes_radius_arquivo`

**Descricao:**
Sincroniza conexoes do Radius para arquivo de auditoria.
No backend PHP:

- Le registros `radacct` pendentes
- Arquiva cada conexao em tabela de historico
- Atualiza status no Radius

### 5. reparar_radius

**Agendamento:** `30 0 * * *` (diario as 00:30)
**Fila RabbitMQ:** `repair_radius_acctstoptime`

**Descricao:**
Repara registros do Radius que estao com `acctstoptime` nulo (sessoes que nao foram finalizadas corretamente).
No backend PHP:

- `del_all_acctstoptime_null()` — remove registros com acctstoptime nulo

### 6. limpeza_logs

**Agendamento:** `30 0 * * *` (diario as 00:30)
**Fila RabbitMQ:** `limpeza_logs`

**Descricao:**
Limpa tabelas de log para evitar acumulo de dados. No backend PHP:

- `sgp_clientes_logs` — logs de clientes
- `sgp_monitor_interfaces_historico` — historico de monitoramento
- `sgp_webservices_cron` — logs do cron
- `radpostauth` — logs de autenticacao Radius
- `SystemEvents` — eventos do sistema

### 7. listar_clientes_vencidos

**Agendamento:** `10 14 * * *` (diario as 14:10)
**Fila RabbitMQ:** `listar_clientes_vencidos`

**Descricao:**
Bloqueia clientes inadimplentes. No backend PHP:

- **Nao executa em fins de semana** (sabado/domingo)
- Busca clientes com contas vencidas
- Bloqueia via RouterOS API
- Envia notificacoes

## Formato do Payload

Todas as tarefas publicam o mesmo formato de mensagem no RabbitMQ:

### Estrutura (antes do Base64)

```json
{
  "gisp_id": 12,
  "gisp_token": "a1b2c3d4e5f6g7h8i9j0",
  "hostname": "177.136.249.51",
  "username": "root",
  "password": "s3nh4_s3cr3t4",
  "database": "gisp_cliente_12"
}
```

### Processo de Codificacao

O PHP original fazia:
```php
$dados = base64_encode(json_encode($g));
```

O Go replica exatamente:
```go
jsonBytes, _ := json.Marshal(payload)
msg := base64.StdEncoding.EncodeToString(jsonBytes)
```

## Comportamento de Resciliencia

Cada publicacao individual possui:

| Estagio | Acao |
|---|---|
| Tentativa 1 | Publica na fila |
| Tentativa 2 | Se falhar, espera 1s e tenta de novo |
| Tentativa 3 | Se falhar, espera 2s e tenta de novo |
| Tentativa 4 | Se falhar, espera 4s e tenta de novo |
| Desistencia | Loga erro e passa para proxima instancia |

Se a conexao com RabbitMQ cair durante a execucao, o `NotifyClose` detecta e reconecta automaticamente em background com backoff de 2s a 60s.
