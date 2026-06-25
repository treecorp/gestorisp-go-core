# HOTFIX-008 — Corrigir panic `slice bounds out of range [:32]` no token do protocolo

**Status:** Implementado
**Autor:** Dev Backend
**Prioridade:** Alta
**Tipo:** Correcao

## 1. Problema

Ao gerar o token para o INSERT em `sgp_clientes_contratos_protocolos`, o codigo usa `[:32]` para truncar a string em 32 caracteres:

```go
token := fmt.Sprintf("tok_%d", rand.Int63())[:32]
```

`rand.Int63()` retorna um inteiro de ate 19 digitos (`9223372036854775807`). O prefixo `"tok_"` tem 4 caracteres. A string resultante tem no maximo **23 caracteres**, mas o `[:32]` tenta acessar indices 0..31 — **panic**.

### 1.1 Stack trace

```
panic: runtime error: slice bounds out of range [:32] with length 23
gestor/internal/pagamento/processar.go:376
gestor/internal/pagamento/criarProtocoloBaixa
```

### 1.2 Ocorrencias

O bug existe em **2 lugares** no arquivo `internal/pagamento/processar.go`:

| Linha | Funcao | Uso |
|-------|--------|-----|
| 376 | `criarProtocoloBaixa` | Token do protocolo de baixa |
| 418 | `desbloquearContratoDB` | Token do protocolo de desbloqueio |

## 2. Correcao

Remover o `[:32]` — o campo `token` em `sgp_clientes_contratos_protocolos` e `varchar(255)` e aceita strings de qualquer tamanho. O PHP usava `md5(uniqid(rand(), true))` com 32 caracteres hex, mas o formato `tok_` + numero tambem e valido.

```go
// ANTES (quebra):
token := fmt.Sprintf("tok_%d", rand.Int63())[:32]

// DEPOIS (funciona):
token := fmt.Sprintf("tok_%d", rand.Int63())
```

### 2.1 Arquivos alterados

| Arquivo | Linhas | Alteracao |
|---------|--------|-----------|
| `internal/pagamento/processar.go` | 376 | Remover `[:32]` |
| `internal/pagamento/processar.go` | 418 | Remover `[:32]` |

## 3. Impacto

- **Corrige o panic** que derruba o worker ao processar pagamento
- Token passa a ter entre 5 (`tok_0`) e 23 (`tok_9223372036854775807`) caracteres
- Nao afeta outras funcionalidades

## 4. Teste

- `go vet ./...` — sem erros
- `go build ./...` — compilacao limpa
- Apos deploy, baixa de fatura nao deve mais panickar
