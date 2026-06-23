
# Gotcha 002: Type assertion em interface no retry

**Data:** 23/06/2026
**Status:** ✅ Corrigido

## Problema

Na primeira versao do `base.go`, a funcao `publicarComRetry` tentou usar uma
interface generica para aceitar qualquer tipo:

```go
func publicarComRetry(rabbit *mensageria.RabbitMQ, fila string,
    instancia interface{ GetID() int }) error {
```

Isso gerou erro de compilacao:
```
cannot use instancia (variable of type interface{GetID() int}) as
dominio.Instancia value in argument to rabbit.PublicarInstancia:
need type assertion
```

## Solucao

Mudar a assinatura para aceitar `dominio.Instancia` diretamente e adicionar
o metodo `GetID()` na struct:

```go
func publicarComRetry(rabbit *mensageria.RabbitMQ, fila string,
    instancia dominio.Instancia) error {
```

## Licao

Go nao faz conversao implicita de interface para tipo concreto.
Se a funcao so trabalha com um tipo especifico, use o tipo diretamente.
Interface generica so faz sentido quando ha multiplos tipos concretos.
