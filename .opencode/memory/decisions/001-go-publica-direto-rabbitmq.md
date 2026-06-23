
# Decisao 001: Go publica direto no RabbitMQ

**Data:** 23/06/2026
**Autor:** opencode
**Status:** ✅ Implementado

## Contexto

O microsservico `gestorisp-ws-cron` (PHP 5.6 + CodeIgniter 3) fazia um `curl` para um
Node.js Producer na porta 3000, que por sua vez publicava a mensagem no RabbitMQ.

Fluxo original:
```
ws-cron (PHP) --curl--> Node.js Producer (3000) --amqp--> RabbitMQ
```

O Node.js Producer (`server.js`) era um pass-through puro — so pegava o
`req.query.dados` (ja base64) e jogava na fila. Nenhuma transformacao,
validacao ou logica de negocios.

## Decisao

O Go deve publicar **direto no RabbitMQ** via protocolo AMQP, eliminando o
Node.js Producer intermediario.

```
Go Cron --amqp--> RabbitMQ
```

## Motivos

1. **Menos latencia:** elimina ~5-10ms de HTTP + abertura de conexao RabbitMQ
2. **Menos componentes:** um servico a menos para manter
3. **Menos pontos de falha:** Node.js Producer nao pode mais cair
4. **Pool de conexoes:** Go mantem conexao persistente, nao abre/fecha por request

## Impacto

- **Positivo:** latencia reduzida, arquitetura mais simples
- **Neutro:** formato dos dados permanece identico (JSON -> Base64)
- **Neutro:** filas RabbitMQ continuam com os mesmos nomes
- **Futuro:** Producer Node.js pode ser descontinuado apos Fase 2

## Arquivos envolvidos
- `internal/infra/mensageria/rabbit.go` — implementacao do publisher AMQP
