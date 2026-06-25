# Decisao 011 — listar_clientes_vencidos publica na fila desconectar_contrato

**Data:** 25/06/2026
**Contexto:** Handler `listar_clientes_vencidos` fazia desconexao direta via RouterOS.

## Problema
O bloqueio de inadimplentes chamava `desconectarCliente()` que abria conexao direta com o RouterOS de cada POP. Isso:
- Duplicava a logica de desconexao (ja existia no handler `desconectar_contrato`)
- Nao tinha retry (se a RB estivesse offline, o cliente nao era desconectado)
- Acoplava o cron `listar_clientes_vencidos` com `routeros` package
- Impedia rastreabilidade (nao passava pela fila)

## Decisao
Remover `desconectarCliente()` de `listar_clientes_vencidos.go` e, em vez disso, publicar uma `MensagemDesconexaoContrato` na fila `desconectar_contrato` (duravel, persistente, retry infinito).

Fluxo novo:
1. Cron publica instancia na fila `listar_clientes_vencidos`
2. Worker processa faturas, bloqueia contratos no banco (TX MySQL)
3. Para cada bloqueado, publica mensagem na fila `desconectar_contrato`
4. Worker `desconectar_contrato` coleta e faz a desconexao no RouterOS (retry infinito)

Para isso, `HandlerListarClientesVencidos` agora recebe `*mensageria.RabbitMQ` como segundo parametro. Em `cmd/worker/main.go`, o registro usa uma closure que captura `rabbit`.

## Consequencias
- Logica de desconexao centralizada no handler `desconectar_contrato` (DRY)
- Retry infinito com backoff na desconexao (se RB offline, tenta de novo)
- Mensagens expiram em 24h (`MensagemDesconexaoContrato.Expirada()`)
- Testes unitarios criados para `HandlerDesconectarContrato` e `Expirada()`
- Validado em producao: PPPoE `04720186475` desconectado com sucesso via fila
