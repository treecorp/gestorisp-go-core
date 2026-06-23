
# Convencao 003: Nomes de pacotes sem underline

**Data:** 23/06/2026
**Status:** ✅ Ativa

## Regra

Nomes de pacotes Go devem ser:
- Em portugues
- Uma unica palavra (sem underlines)
- Letras minusculas

## Exemplos

```go
// Correto
package banco       // em vez de package database
package mensageria  // em vez de package message_queue
package logger      // em vez de package log_util

// Incorreto
package meu_pacote
package database_utils
```

## Excecoes

Nao ha excecoes. Se precisar de mais de uma palavra, use a convencao Go de
abreviar ou encontrar um nome curto que represente o conceito.

## Motivo

- Padrao idiomatico Go: pacotes sao nomes curtos e sem underlines
- `go vet` emite avisos para nomes com underline
- Consistencia com a biblioteca padrao
