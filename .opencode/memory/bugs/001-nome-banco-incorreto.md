
# Bug 001: Nome do banco incorreto (gispadm vs gisp_adm)

**Data:** 23/06/2026
**Status:** ✅ Corrigido
**Severidade:** Alta (impedia o sistema de iniciar)

## Sintoma

```
2026/06/23 15:07:48 Falha ao conectar no banco: falha ao pingar banco:
Error 1049 (42000): Unknown database 'gispadm'
```

## Causa

O nome real do banco GISPADM e `gisp_adm` (com underline), mas estava
configurado como `gispadm` (sem underline) em 3 lugares:
- `internal/config/config.go` — fallback hardcoded
- `Dockerfile` — ENV hardcoded
- `.env.exemplo` — template

## Diagnostico

O usuario informou "o nome do banco e gisp_adm" durante a sessao. O erro
1049 e claro: o banco `gispadm` nao existe no servidor MySQL.

## Solucao

Alterado `gispadm` para `gisp_adm` em:
1. `internal/config/config.go:35`
2. `Dockerfile:31`
3. `.env.exemplo:14`

## Licoes

- Sempre confirmar os nomes exatos dos recursos com o usuario/operacao
- O fallback hardcoded no config.go ajudou a identificar rapidamente pois
  permitiu rodar sem variavel de ambiente
