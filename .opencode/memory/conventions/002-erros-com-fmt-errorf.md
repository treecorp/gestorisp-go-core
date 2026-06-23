
# Convencao 002: Erros com fmt.Errorf

**Data:** 23/06/2026
**Status:** ✅ Ativa

## Regra

Todos os erros devem ser propagados usando `fmt.Errorf` com o verbo `%w`
para preservar a cadeia de erros (error wrapping).

## Exemplos

```go
// Correto
return nil, fmt.Errorf("falha ao conectar no banco: %w", err)

// Incorreto
return nil, errors.New("falha ao conectar: " + err.Error())
```

## Motivo

- `errors.Is()` e `errors.As()` funcionam corretamente com `%w`
- A pilha de erros e preservada para debug
- Padrao idiomatico em Go

## Onde se aplica

- Todos os pacotes em `internal/`
- Funcoes que retornam `error`
