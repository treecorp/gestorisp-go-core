# SDD-015 — Gateway Assincrono + Worker Desconexao RouterOS

**Status:** Proposta
**Autor:** Dev Backend
**Prioridade:** Alta
**Dependencias:** SDD-014, infra existente (`banco`, `mensageria`, `fuso`, `logger`, `routeros`)

## 1. Objetivo

Refatorar o gateway de pagamentos (SDD-014) de **sincrono** para **assincrono** via RabbitMQ:

- Gateway vira **apenas publicador** (recebe webhook, valida, publica na fila, retorna 200)
- Novo **Worker Pagamento** processa a baixa com **transacao MySQL** e **retry ate 5x**
- Novo **Worker Desconexao** desconecta o cliente do RouterOS com **retry infinito**
- Fila `desconectar_contrato` reutilizavel por outros workers (listar_clientes_vencidos, cron_1, etc.)

## 2. Arquitetura

```
Iugu ──POST──▶ Gateway (:8082)
               ├── Autenticar token
               ├── Parse webhook
               └── Publicar "processar_pagamento_iugu"
               └── Return 200

[RabbitMQ] ←─ processar_pagamento_iugu (duravel, persistente)

Worker Pagamento
  ├── Decodificar MensagemPagamentoIugu
  ├── Conectar DB instancia
  ├── Verificar duplicata (gisp_iugu_gatilhos)
  ├── BEGIN TX
  │    ├── Consultar Iugu API
  │    ├── UPDATE sgp_clientes_faturas SET status='Pago'
  │    ├── INSERT gisp_iugu_faturas_json
  │    ├── lancarCaixa()
  │    ├── criarProtocoloBaixa()
  │    ├── UPDATE contrato SET status='Ativo'
  │    ├── DELETE radreply WHERE value='pgcorte'
  │    └── INSERT protocolo desbloqueio
  ├── COMMIT
  ├── Publicar "desconectar_contrato"
  ├── Ack
  └── Erro: tentativa++ < 5 → Nack(true) | >= 5 → Nack(false) + log critico

[RabbitMQ] ←─ desconectar_contrato (duravel, persistente)

Worker Desconexao
  ├── Decodificar MensagemDesconexaoContrato
  ├── Conectar RouterOS
  ├── Verificar usuario ativo
  │    ├── Ativo → Desconectar → Ack
  │    ├── Inativo → Ack (ja desconectado)
  │    └── Erro router → Nack(true) retry infinito
```

## 3. Mensageria

### 3.1 Filas

| Fila | Duravel | Delivery Mode | Retry | Descricao |
|------|---------|---------------|-------|-----------|
| `processar_pagamento_iugu` | `true` | `Persistent (2)` | Max 5 tentativas | Processa pagamento Iugu com TX |
| `desconectar_contrato` | `true` | `Persistent (2)` | Infinito | Desconecta cliente do RouterOS |

### 3.2 Mensagens

```go
// internal/dominio/pagamento.go
type MensagemPagamentoIugu struct {
    Instancia Instancia         `json:"instancia"`
    Event     string            `json:"event"`
    Data      map[string]string `json:"data"`
    Tentativa int               `json:"tentativa"`
}

// internal/dominio/desconexao.go
type MensagemDesconexaoContrato struct {
    Instancia   Instancia `json:"instancia"`
    ContratoID  int       `json:"contrato_id"`
    ClienteNome string    `json:"cliente_nome"`
    PPPoEUser   string    `json:"pppoe_user"`
    PopIPv4     string    `json:"pop_ipv4"`
    PopPort     string    `json:"pop_port"`
    PopUser     string    `json:"pop_user"`
    PopPass     string    `json:"pop_pass"`
}
```

### 3.3 Novo metodo no RabbitMQ

`PublicarMensagem(fila string, payload interface{}) error`
- JSON marshal → Base64 encode → Publicar
- `QueueDeclare` com `durable=true`
- `Publishing` com `DeliveryMode = amqp.Persistent`

## 4. Pacote pagamento (extraido do gateway)

```go
// internal/pagamento/processar.go
package pagamento

type ResultadoBaixa struct {
    ContratoID    int
    ClienteNome   string
    PPPoEUser     string
    PopIP         string
    PopPort       string
    PopUser       string
    PopPass       string
    PrecisaDesconectar bool
}

func ProcessarPagamento(db *sql.DB, instancia dominio.Instancia,
    data map[string]string, iuguFaturaID string, statusEsperado string) (*ResultadoBaixa, error)
```

A funcao `ProcessarPagamento`:
1. Executa toda a logica de `executarBaixa()` + `processarPagamentoIuguDireto()` / `processarPagamentoJuno()` / `processarPagamentoExternal()`
2. Envolve em `BEGIN` / `COMMIT` / `ROLLBACK`
3. Retorna `ResultadoBaixa` para o worker publicar na fila de desconexao

### 4.1 Transacao

```
BEGIN TX
  ├── marcarProcessando()               UPDATE gisp_iugu_gatilhos
  ├── Consultar Iugu API                GET /v1/invoices/{id}
  ├── UPDATE sgp_clientes_faturas       SET status='Pago'
  ├── marcarProcessado()                UPDATE gisp_iugu_gatilhos
  ├── INSERT gisp_iugu_faturas_json
  ├── lancarCaixa()                     UPDATE caixa + INSERT fluxo
  ├── criarProtocoloBaixa()             INSERT protocolo baixa
  ├── desbloquearContratoDB()           UPDATE contrato + DELETE radreply + INSERT protocolo
COMMIT

Fora da TX:
  └── Se desbloqueado → publicar "desconectar_contrato"
```

Se qualquer passo falhar → `ROLLBACK` → erro retorna para o worker que decide retry.

## 5. Gateway modificado

### gateway/server.go
```go
type Servidor struct {
    cfg     *config.Config
    servico *http.Server
    rabbit  *mensageria.RabbitMQ
}

func NovoServidor(cfg *config.Config, rabbit *mensageria.RabbitMQ) *Servidor
```

### gateway/iugu_webhook.go
- Remove `db *sql.DB` — gateway nao conecta mais no banco da instancia
- `HandleWebhook` apenas: parse form → publicar na fila → return 200
- Remove `handleStatusChanged`, `codigosOrigem`, `truncate`, `gerarProtocolo`, `idCounter`, `origemPagamento`, `limparNumero` — movidos para pagamento

## 6. Workers

### 6.1 Worker Pagamento (`internal/worker/processar_pagamento_iugu.go`)

```go
func HandlerProcessarPagamentoIugu(body []byte) error {
    // 1. Decodificar base64 → JSON → MensagemPagamentoIugu
    // 2. Conectar DB instancia
    // 3. Chamar pagamento.ProcessarPagamento(db, ...)
    // 4. Se ResultadoBaixa.PrecisaDesconectar:
    //      Publicar "desconectar_contrato" com dados do POP
    // 5. Ack
}
```

### 6.2 Worker Desconexao (`internal/worker/desconectar_contrato.go`)

```go
func HandlerDesconectarContrato(body []byte) error {
    // 1. Decodificar → MensagemDesconexaoContrato
    // 2. Conectar RouterOS (go-routeros/v3)
    // 3. Verificar ativo
    //    - Se ativo: Desconectar
    //    - Se inativo: log + nil (ja desconectado)
    // 4. Sucesso → nil (Ack)
    // 5. Falha → error (Nack true, retry infinito)
}
```

Este worker **nao acessa banco de dados** — todos os dados necessarios ja estao na mensagem.

## 7. Workers: consumo com suporte a retry

O `Consumidor` atual usa `handler func(dominio.Instancia) error`. Os novos handlers precisam receber `[]byte` para decodificar mensagens customizadas.

### worker/consumidor_mensagem.go (NOVO)
```go
type ConsumidorMensagem struct {
    Fila    string
    Handler func([]byte) error
}
```

### worker/worker.go (MODIFICADO)
Adicionar:
- `IniciarMensagem(consumidores []ConsumidorMensagem)` — mesmo loop de `Iniciar` mas com `consumirMensagem()`
- `consumirMensagem(cons ConsumidorMensagem)` — igual `consumir()` mas chama `processarMensagemGenerica()`
- `processarMensagemGenerica(tag string, msg amqp.Delivery, handler func([]byte) error)`:
  - Chama `handler(msg.Body)`
  - Se erro: verifica se `tentativa` no JSON < 5 → Nack(true) | Nack(false)
  - Se sucesso: Ack

Para `desconectar_contrato` (retry infinito): o handler sempre retorna erro se RouterOS falhar, nunca incrementa tentativa. O `processarMensagemGenerica` trata filas sem campo `tentativa` como retry infinito (sempre Nack(true)).

## 8. Arquivos alterados

| # | Arquivo | Acao |
|---|---------|------|
| 1 | `internal/dominio/pagamento.go` | **Criar** |
| 2 | `internal/dominio/desconexao.go` | **Criar** |
| 3 | `internal/infra/mensageria/rabbit.go` | **Modificar** — add `PublicarMensagem()` |
| 4 | `internal/pagamento/processar.go` | **Criar** — logica extraida + TX |
| 5 | `internal/pagamento/cliente_iugu.go` | **Criar** — HTTP client extraido |
| 6 | `internal/gateway/server.go` | **Modificar** — injetar rabbit |
| 7 | `internal/gateway/iugu_webhook.go` | **Modificar** — so publica na fila |
| 8 | `internal/gateway/iugu_pagamento.go` | **Remover** (movido p/ pagamento/) |
| 9 | `internal/gateway/cliente_iugu.go` | **Remover** (movido p/ pagamento/) |
| 10 | `internal/worker/consumidor_mensagem.go` | **Criar** |
| 11 | `internal/worker/worker.go` | **Modificar** — add consumo generico |
| 12 | `internal/worker/processar_pagamento_iugu.go` | **Criar** |
| 13 | `internal/worker/desconectar_contrato.go` | **Criar** |
| 14 | `cmd/gateway/main.go` | **Modificar** — conectar RMQ |
| 15 | `cmd/worker/main.go` | **Modificar** — registrar 2 consumidores |

## 9. Impacto em outros workers

A fila `desconectar_contrato` e **reutilizavel**:

| Worker atual | Onde chama RouterOS direto | Futuro: publica na fila |
|-------------|---------------------------|------------------------|
| `listar_clientes_vencidos.go` | `desconectarCliente()` com RouterOS | Publica `desconectar_contrato` |
| `cron_1.go` | `desbloquearUsuariosTravados()` com RouterOS | Publica `desconectar_contrato` |

Nenhuma alteracao no worker de desconexao — a mensagem ja contem todos os dados.

## 10. Observabilidade

- Logs dos workers seguem padrao existente: `[tag] level mensagem`
- Erros de pagamento apos 5 tentativas: log critico com `logger.Erro`
- Erros de desconexao: log `logger.Aviso` a cada tentativa + `logger.Sucesso` quando conseguir
- Gatilhos nao processados ficam com `gisp_exec='0'` e `gisp_exec_status=Erro X` no banco (rastreavel)
