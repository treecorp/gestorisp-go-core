# Gotcha: Worker deve usar prefetch 1 (uma mensagem por vez)

## Contexto
Os workers Go estavam consumindo todas as mensagens da fila de uma vez,
sobrecarregando o banco e processamento. O comportamento atual correto
(igual ao worker JS legado) e consumir 1 mensagem por vez.

## Código original (Node.js — amqplib)
```js
ch.prefetch(1);
```
Presente em TODOS os workers legados: `worker.js`, `worker2.js`,
`worker_cron_1.js`, `worker_radius.js`, etc.

## Código correto (Go — streadway/amqp)
```go
err = canal.Qos(1, 0, false)
if err != nil {
    logger.Erro(tag, "Falha ao setar prefetch 1: %v. Reintentando em 5s...", err)
    w.fecharCanal(canal)
    time.Sleep(5 * time.Second)
    continue
}
```

Deve ser chamado APOS `QueueDeclare` e ANTES de `Consume`.

## Referencia
- `internal/worker/worker.go` — linha 67 (no metodo `consumir`)
- Workers JS originais em `RabbitMQ/gisp-rabbitmq/app/src/worker*.js`

## Afeta
- Todos os consumidores registrados no worker Go
- Fila `check_pop_status`, `run_cluster`, `sync_conexoes_radius_arquivo`
