# SDD-010 — Handler listar_clientes_vencidos

**Status:** Aprovado
**Autor:** Dev Backend
**Prioridade:** Alta
**Dependencias:** go-routeros/v3 (existente), internal/infra/fuso (existente)

## 1. Objetivo
Bloqueio automatico de contratos inadimplentes. Roda diariamente as 14:10.
Nao executa em fins de semana (sabado/domingo).

## 2. Comportamento Esperado
- Verifica dia da semana; se sabado(6) ou domingo(0), loga e retorna (sem erro)
- Conecta na instancia
- Le `sgp_parametros.dias_bloqueio` (global, default 5)
- Seleciona todas as faturas vencidas (`status='Pendente'`, `isento='Nao'`, vencimento entre `2018-06-01` e ontem)
- Para cada fatura, em **transacao**:
  1. Le contrato (`id, token, status, pppoe_user, dias_bloqueio, permitir_bloqueio`)
  2. Le desbloqueio de confianca ativo
  3. Validacoes em ordem:
     - `permitir_bloqueio != 0`? Se `0`, pula (log + continue)
     - `status == 'Ativo'`? Se nao, pula
     - Sem trust-unblock OU hoje >= `data_hora_bloqueio`? Se nao, pula
     - `dias_atraso > dias_bloqueio`? (usa contrato.dias_bloqueio se preenchido, senao global) — se nao, pula
  4. Acoes de bloqueio:
     - Desativa trust-unblock (se existir, `status='Inativo'`)
     - INSERT em `radreply` (Framed-Pool=pgcorte) se nao existir
     - UPDATE `sgp_clientes_contratos`: `status='Bloqueado'`, `data_hora_bloqueio=agora`
     - INSERT em `sgp_clientes_contratos_protocolos` (log do bloqueio)
     - UPDATE `sgp_clientes_contratos_conexoes`: `data_desconexao`, `hora_desconexao`, `status='Offline'`
     - UPDATE `sgp_clientes_contratos`: `conexao='Offline'`, `data_ultima_conexao_atividade`, ...
     - `adicionarLogDesconexao()` (reuso de cron_1.go)
     - Acumula em `[]clienteBloqueado` (pppoe_user, pop_id)
- Apos processar todas as faturas, para cada clienteBloqueado:
  - Carrega POPs (`carregarPops()`, reuso de cron_1.go)
  - Conecta no RouterOS via go-routeros
  - Verifica se sessao PPPoE esta ativa
  - Se ativa, remove (`/ppp/active/remove`)
  - Loga erros mas continua (nao bloqueia o lote)

## 3. Precedencia de `dias_bloqueio`

| Prioridade | Origem | Campo | Exemplo |
|------------|--------|-------|---------|
| 1º | Contrato (se NOT NULL) | `sgp_clientes_contratos.dias_bloqueio` | 3 |
| 2º | Parametro global | `sgp_parametros.dias_bloqueio` | 5 |

## 4. Contratos

### Handler
```go
func HandlerListarClientesVencidos(instancia dominio.Instancia) error
```

### Structs
```go
type faturaVencida struct {
    ID            int
    ContratoID    int
    ContratoToken string
    ClienteToken  string
    PPPoEUser     string
    PopID         int
    Vencimento    string
}

type clienteBloqueado struct {
    PPPoEUser string
    PopID     int
}

type parametrosSistema struct {
    DiasBloqueio int
}
```

### Fila RabbitMQ
`listar_clientes_vencidos` (ja declarada no cron em `cmd/gestor/main.go`)

### Cron
`0 10 14 * * *` — diariamente 14:10 (verificacao de fim de semana dentro do handler)

## 5. SQL

### Parametros
```sql
SELECT dias_bloqueio FROM sgp_parametros LIMIT 1
```

### Faturas vencidas
```sql
SELECT f.id, f.contrato_id, f.contrato_token, f.cliente_token,
       c.pppoe_user, COALESCE(c.pop_id, 0) AS pop_id,
       DATE_FORMAT(f.vencimento, '%Y-%m-%d') AS vencimento
FROM sgp_clientes_faturas f
INNER JOIN sgp_clientes_contratos c ON c.id = f.contrato_id
WHERE c.isento = 'Nao'
  AND f.status = 'Pendente'
  AND f.vencimento BETWEEN '2018-06-01' AND CURDATE()
  AND f.vencimento < CURDATE()
ORDER BY f.vencimento ASC
```

### Contrato
```sql
SELECT id, token, status, pppoe_user,
       permitir_bloqueio,
       CAST(dias_bloqueio AS UNSIGNED) AS dias_bloqueio
FROM sgp_clientes_contratos WHERE id = ?
```

### Desbloqueio confianca
```sql
SELECT id, data_hora_bloqueio
FROM sgp_clientes_contratos_desbloqueio_confianca
WHERE contrato_id = ? AND status = 'Ativo'
ORDER BY id DESC LIMIT 1
```

### Radreply check
```sql
SELECT id FROM radreply
WHERE attribute = 'Framed-Pool' AND value = 'pgcorte' AND username = ?
LIMIT 1
```

### INSERT radreply
```sql
INSERT INTO radreply
(username, attribute, op, value, sgp_cliente_token, sgp_contrato_token, sgp_contrato_id)
VALUES (?, 'Framed-Pool', '=', 'pgcorte', ?, ?, ?)
```

### UPDATE contrato status
```sql
UPDATE sgp_clientes_contratos
SET status = 'Bloqueado', data_hora_bloqueio = ?
WHERE id = ?
```

### INSERT protocolo
```sql
INSERT INTO sgp_clientes_contratos_protocolos
(token, contrato_id, contrato_token, protocolo, data_hora, descricao,
 titulo, dados_antigos, user_id, user_nome)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 'Robot')
```

### Ultima conexao
```sql
SELECT token FROM sgp_clientes_contratos_conexoes
WHERE contrato_token = ?
ORDER BY id DESC LIMIT 1
```

### UPDATE conexao
```sql
UPDATE sgp_clientes_contratos_conexoes
SET data_desconexao = ?, hora_desconexao = ?, status = 'Offline'
WHERE token = ?
```

### UPDATE contrato conexao
```sql
UPDATE sgp_clientes_contratos
SET conexao = 'Offline',
    data_ultima_conexao_atividade = ?,
    hora_ultima_conexao_atividade = ?,
    data_hora_ultima_conexao_atividade = ?
WHERE id = ?
```

### Desativar trust-unblock
```sql
UPDATE sgp_clientes_contratos_desbloqueio_confianca
SET status = 'Inativo'
WHERE id = ?
```

## 6. Tratamento de Erros
- Fim de semana → loga e retorna `nil` (sem erro, sem Nack)
- Cada fatura processada em transacao: erro em uma nao afeta as demais
- RouterOS offline → loga (`logger.Aviso`) e continua para proximo cliente
- Se nenhum cliente bloqueado → loga e retorna `nil`
- Falha de conexao com banco → retorna erro → worker Nack

## 7. Registro no Worker

Incluir no slice de `Consumidor` em `cmd/worker/main.go`:
```go
{
    Fila:    "listar_clientes_vencidos",
    Handler: worker.HandlerListarClientesVencidos,
},
```

## 8. Codigo Reutilizado

| Funcao | Origem | Uso |
|--------|--------|-----|
| `gerarToken()` | `cron_1.go` | Token do protocolo |
| `adicionarLogDesconexao()` | `cron_1.go` | Log de desconexao |
| `carregarPops()` | `cron_1.go` | Mapa de POPs |
| `routeros.Conectar()` | `client.go` | Conexao RouterOS |
| `routeros.VerificarUsuarioAtivo()` | `client.go` | Checa sessao ativa |
| `routeros.DesconectarUsuario()` | `client.go` | Remove sessao |

## 9. Estimativa
~230 linhas, 1 arquivo novo (`internal/worker/listar_clientes_vencidos.go`), +3 linhas `cmd/worker/main.go`
