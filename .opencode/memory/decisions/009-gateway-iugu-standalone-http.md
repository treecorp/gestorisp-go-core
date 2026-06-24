# Decisao 009 — Gateway Iugu como binario HTTP standalone

**Data:** 24/06/2026
**Contexto:** Port do webhook Iugu do PHP (CodeIgniter) para Go.

## Decisao
Criar um binario separado `cmd/gateway` (servidor HTTP standalone, porta 8082)
em vez de embutir o servidor HTTP no worker ou no cron.

## Motivos
- Gateway e puramente HTTP (webhooks), nao usa RabbitMQ
- Escala independentemente do cron e do worker
- Entrypoint `SERVICO=gateway` no Dockerfile unificado

## Alternativa descartada
- Embutir no worker: acoplaria responsabilidades diferentes (consumidor RabbitMQ + servidor HTTP)
- Embutir no cron: cron nao deveria expor porta HTTP

## Impacto
- `go vet ./...` e `go build ./...` continuam passando
- Nenhuma alteracao nos binarios existentes (gestor, worker, dashboard)
- +1 servico no Dockerfile
