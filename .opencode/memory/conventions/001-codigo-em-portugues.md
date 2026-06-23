
# Convencao 001: Codigo e comentarios em portugues

**Data:** 23/06/2026
**Status:** ✅ Ativa

## Regra

Todo o codigo-fonte, comentarios, nomes de identificadores (variaveis,
funcoes, pacotes, tipos, arquivos) e documentos do projeto devem estar em
**portugues brasileiro**.

## Excecoes

- Palavras-chave da linguagem Go (if, for, func, return, etc.)
- Nomes de bibliotecas externas
- Chamadas de API externa (ex: `QueueDeclare`, `Publish`)
- Acronimos consagrados (URL, HTTP, JSON, AMQP, SQL, DB)

## Exemplos

```go
// Correto
func BuscarInstanciasAtivas() ([]dominio.Instancia, error) {
    return pool.Query(query)
}

// Incorreto
func GetActiveInstances() ([]dominio.Instancia, error) {
    return pool.Query(query)
}
```

## Motivo

O usuario explicitamente solicitou: "quero que seja em portugues o codigo,
documentado". Facilita o entendimento para a equipe que mantem o sistema.
