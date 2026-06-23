
# Padrao 002: Retry com backoff exponencial

**Data:** 23/06/2026
**Status:** ✅ Ativo

## Descricao

Todas as operacoes que podem falhar transitoriamente (conexao, publicacao)
usam retry com backoff exponencial.

Duas variantes:

### 1. Loop infinito (conexoes)

Usado para conexoes iniciais (banco, RabbitMQ) e reconexao durante execucao.

```go
espera := 2 * time.Second
for {
    err := tentarConectar()
    if err == nil {
        return
    }
    logger.Aviso("...", err, espera)
    time.Sleep(espera)
    espera *= 2
    if espera > 60*time.Second {
        espera = 60 * time.Second  // cap
    }
}
```

### 2. Tentativas limitadas (por instancia)

Usado para publicacao de cada instancia na fila. Nao pode travar o lote todo.

```go
tentativas := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
for i, espera := range tentativas {
    err := publicar()
    if err == nil {
        return nil
    }
    if i < len(tentativas)-1 {
        time.Sleep(espera)  // continua tentando
    }
}
return ultimoErro  // desiste, passa para proxima instancia
```

## Onde se aplica

| Local | Tipo | Maximo |
|---|---|---|
| `banco.ConectarComRetry` | Loop infinito | 60s entre tentativas |
| `rabbit.ConectarComRetry` | Loop infinito | 60s entre tentativas |
| `rabbit.reconectar` | Loop infinito | 60s entre tentativas |
| `base.publicarComRetry` | 3 tentativas | 4s ultima espera |

## Quando usar

- Toda operacao de IO que pode falhar transitoriamente
- Conexoes de rede (banco, mensageria, API externa)
- Publicacao de mensagens
- Operacoes em lote onde uma falha nao deve bloquear as demais
