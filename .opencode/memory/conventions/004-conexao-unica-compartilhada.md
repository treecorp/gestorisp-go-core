
# Convencao 004: Conexao unica compartilhada

**Data:** 23/06/2026
**Status:** ✅ Ativa

## Regra

Conexoes com banco de dados e RabbitMQ devem ser **inicializadas uma unica
vez** no `main.go` e **compartilhadas** entre todas as tarefas.

Nao abrir uma conexao nova por job.

## Fluxo

```
main.go
  -> banco.ConectarComRetry()    <- 1 pool MySQL
  -> mensageria.Conectar()       <- 1 conexao RabbitMQ
  -> cron.NovoAgendador(db, rabbit)  <- injeta nos jobs
```

## Motivo

- **Performance:** evita abrir/fechar conexao a cada execucao (como no PHP)
- **Recursos:** uma conexao RabbitMQ com canal unico vs N conexoes por job
- **Monitoramento:** `NotifyClose` unico para detectar queda
- **Pool:** MySQL com 10 conexoes max, reutilizaveis entre goroutines

## Onde se aplica

- `internal/cron/agendador.go` recebe `rabbit` ja conectado
- `internal/cron/tarefas/base.go` usa o `rabbit` injetado
- Nao criar `Conectar()` dentro de jobs individuais
