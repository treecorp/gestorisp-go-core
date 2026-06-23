
# Gotcha 001: Porta RabbitMQ nao padrao 31837

**Data:** 23/06/2026
**Status:** ✅ Documentado

## Problema

A porta do RabbitMQ fornecida foi `31837`, que **nao e a porta padrao** (5672).

## Impacto

Se alguem tentar usar a config padrao `RABBITMQ_PORT=5672` (ou nao definir
a env), a conexao falhara silenciosamente.

## Licao

Sempre confirmar as portas exatas com o usuario/operacao antes de assumir
valores padrao.

A config atual usa fallback correto:
```go
Porta: obterEnv("RABBITMQ_PORT", "31837"),
```

## Como evitar

- Nao assumir portas padrao (5672, 3306, 80)
- Sempre documentar as portas usadas em `.opencode/memory/`
- Usar `.env.exemplo` com valores reais para ambiente de desenvolvimento
