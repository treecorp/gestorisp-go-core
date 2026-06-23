
# Decisao 004: Estrutura modular para comportar fases futuras

**Data:** 23/06/2026
**Autor:** opencode
**Status:** ✅ Implementado

## Contexto

O projeto substituira 5 microsservicos em Go:
1. `gestorisp-ws-cron` — agendador (Fase 1 - ATUAL)
2. `gestorisp-ws-rabbimq` — producer (Fase 1)
3. Workers RabbitMQ em Node.js (Fase 2)
4. `gestorisp-ws-gateway-pagamentos` — pagamentos (Fase 3)
5. `gestorisp-so` — sistema operacional (Fase 3)

Precisavamos de uma estrutura que comportasse todas essas fases sem
refatoracoes grandes no futuro.

## Decisao

Adotar uma estrutura modular com separacao clara por responsabilidade:

```
internal/
  config/         <- Configuracao central (env vars)
  dominio/        <- Entidades de negocio (Instancia, Assinante, Contrato, ...)
  infra/          <- Infraestrutura (banco, mensageria, logger, http, ...)
  cron/           <- Agendador de tarefas (Fase 1)
  worker/         <- Consumidores RabbitMQ (Fase 2)
  api/            <- API HTTP (Fase 3)
```

## Motivos

1. **Crescimento organico:** cada fase adiciona uma pasta nova sem quebrar
   as existentes
2. **Separation of concerns:** dominio puro separado de infraestrutura
3. **Facil de testar:** cada pacote tem responsabilidade unica
4. **Facil de entender:** um novo desenvolvedor ja sabe onde encontrar cada
   coisa pela estrutura

## Impacto

- Fase 2 (workers): criar `internal/worker/` e `internal/dominio/` com novas entidades
- Fase 3 (API): criar `internal/api/` com handlers e middleware
- Infraestrutura compartilhada (banco, mensageria, logger) ja esta pronta

## Arquivos envolvidos
- Estrutura completa do projeto
