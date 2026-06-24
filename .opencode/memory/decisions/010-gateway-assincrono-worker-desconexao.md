# Decisao 010 — Gateway assincrono via RabbitMQ + Worker Desconexao

**Data:** 2026-06-24
**SDD:** SDD-015
**Status:** Implementado

## Contexto

O gateway de pagamentos (SDD-014) processava webhooks Iugu de forma sincrona:
recebia o POST, conectava no banco da instancia, consultava Iugu API,
atualizava tudo e so entao retornava 200. Isso tornava o gateway lento
e vulneravel a timeouts do Iugu.

## Decisao

Converter o gateway para assincrono:

1. Gateway vira **apenas publicador** RabbitMQ — valida token, publica
   na fila `processar_pagamento_iugu`, retorna 200 imediatamente
2. **Worker Pagamento** consome da fila, executa a baixa com **transacao
   MySQL** e retry maximo de 5 tentativas
3. **Worker Desconexao** consome da fila `desconectar_contrato`, conecta
   no RouterOS e desconecta o cliente com **retry infinito**

## Detalhes

- Filas `processar_pagamento_iugu` e `desconectar_contrato`: **duravel**
  (`durable=true`) com **entrega persistente** (`DeliveryMode=Persistent`)
- `mensageria/rabbit.go`: novo metodo `PublicarMensagem()` generico
- `internal/pagamento/`: novo pacote com a logica extraida do gateway
- `internal/gateway/iugu_webhook.go`: simplificado — so publica na fila
- `internal/worker/worker.go`: add `IniciarMensagem()` + `processarMensagemGenerica()`
  com logica de retry baseada em campo `tentativa` no JSON
- Transacao MySQL no `executarBaixa()`: BEGIN/COMMIT/ROLLBACK
- Apos COMMIT, se o contrato foi desbloqueado, publica na fila de desconexao
- Worker de desconexao **nao acessa banco de dados** — todos os dados
  necessarios (IP POP, usuario PPPoE, credenciais) estao na mensagem

## Arquivos alterados

14 arquivos entre criacao, modificacao e remocao.

## Consequencias

- Gateway responde 200 em milissegundos (antes levava segundos)
- Baixa de pagamento com consistencia transacional
- Desconexao RouterOS com retry infinito ate o equipamento responder
- Fila `desconectar_contrato` reutilizavel por outros workers
  (`listar_clientes_vencidos`, `cron_1`)
