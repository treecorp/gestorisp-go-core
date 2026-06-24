# HOTFIX-002 — Remover fluxo Juno do processador de pagamento Iugu

**Status:** Aprovado
**Autor:** Dev Backend
**Prioridade:** Baixa
**Tipo:** Limpeza

## 1. Problema

O pacote `internal/pagamento/processar.go` contém a função `processarJuno()`
que tratava faturas criadas no Juno e importadas para o Iugu.

Este fluxo não existe mais — toda fatura passa diretamente pelo Iugu.
O código morto adiciona complexidade desnecessária.

## 2. O que será removido

| Item | Local | Linhas |
|------|-------|--------|
| Função `processarJuno()` | `processar.go` | ~30 |
| Dispatch `if len(externalRef) == 9` | `ProcessarPagamento()` | ~5 |
| Parâmetro `observacao` "JUNO..." | `executarBaixa()` | ~2 |

## 3. Assinatura final de `ProcessarPagamento`

```go
func ProcessarPagamento(db *sql.DB, instancia dominio.Instancia,
    data map[string]string, iuguFaturaID string, statusEsperado string) (*ResultadoBaixa, error) {

    // validações...
    if statusEsperado == "externally_paid" {
        return processarExternal(db, instancia, data, iuguFaturaID, externalRef)
    }
    return processarIuguDireto(db, instancia, data, iuguFaturaID, statusEsperado, externalRef)
}
```

## 4. Impacto

- `executarBaixa()` deixa de receber `observacao` (sempre `""`)
- Demais funções inalteradas: `processarIuguDireto`, `processarExternal`,
  `executarBaixa`, TX, caixa, protocolo, desbloqueio, helpers
- Workers, gateway, mensageria: **nenhuma alteração**
