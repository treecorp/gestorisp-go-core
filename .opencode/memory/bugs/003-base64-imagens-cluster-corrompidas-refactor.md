# Bug 003 — base64 de imgOffline/imgOnline corrompidas na refatoracao v2

**Status:** Corrigido (HOTFIX-009)

## Descricao

Durante a refatoracao estrutural (commit `6d40601`), as constantes base64 de
`imgOffline` e `imgOnline` em `internal/handler/worker/run_cluster.go` foram
alteradas acidentalmente. Apenas 1 caractere no meio da string base64 diferenciava
cada imagem, mas isso gerava pixels visivelmente diferentes no dashboard.

| Imagem | main (correto) | v2 (bugado) |
|--------|----------------|-------------|
| `imgOffline` | `ARMRIUISJB...` | `AREhMUISJB...` (indice 331) |
| `imgOnline` | `RkETITH...` | `RESEkMTL...` (indice 332) |

As outras imagens (`imgBloqueado`, `imgDesconexao`, `imgPendenteAtivacao`)
permaneceram identicas.

## Sintoma

Contratos offline apareciam com cor/icone errado no cluster do dashboard.
Contratos online tambem exibiam imagem incorreta.

## Causa raiz

O arquivo foi recriado manualmente durante a refatoracao e as strings base64
foram transcritas com erro. Como sao strings longas (600-700 chars), o erro
passou despercebido em code review.

## Correcão

Copiado o valor exato de `main` (`232db16`) para ambas as constantes em
`internal/handler/worker/run_cluster.go`.

## Arquivo afetado

`internal/handler/worker/run_cluster.go` (constantes `imgOffline`, `imgOnline`)

## Licao aprendida

Sempre comparar strings longas (base64, hashes) com diff/cmp entre branches
ao refatorar arquivos que as contenham. Validacao visual tambem e necessaria.
