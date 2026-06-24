# Gotcha: POP pode ficar offline entre o check_pop_status e o cron_1

## Contexto
O `desbloqueia_user_bloqueado_no_banco` depende do status do POP para decidir
se deve ou nao verificar a sessao do usuario na RouterBoard.

## Fluxo problematico
```
T+0s  → check_pop_status: POP X online → sgp_pops.status = "OPERACIONAL"
T+15s → POP X cai (energia, link, reboot)
T+20s → cron_1 executa: le sgp_pops.status = "OPERACIONAL" (dado velho 20s)
T+20s → cron_1 tenta conectar na RB do POP X → FALHA
```

## Comportamento seguro do Go
```go
conn, err := routeros.Conectar(...)
if err != nil {
    logger.Aviso(tag, "POP %d inacessivel: %v - preservando sessao %s", ...)
    continue // NAO fecha sessao RADIUS
}
```

**NUNCA fechar sessao RADIUS sem confirmacao da RouterBoard.**
O pior caso: sessao travada espera ate 5 min (proxima execucao do cron_1).

## Referencia
- `internal/worker/cron_1.go` — funcao `desbloquearUsuariosTravados`

## Afeta
- Worker `cron_1` — desbloqueia_user_bloqueado_no_banco
- Apenas quando POP oscila entre a checagem de 30s e a execucao do cron
