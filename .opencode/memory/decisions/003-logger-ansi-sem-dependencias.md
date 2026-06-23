
# Decisao 003: Logger com ANSI puro sem dependencias

**Data:** 23/06/2026
**Autor:** opencode
**Status:** ✅ Implementado

## Contexto

O projeto usa `log.Println` e `log.Printf` para todas as mensagens. O usuario
pediu: "formate bem bonito o console, bem organizado e colorido, registrando
os processos do cron, a execucao e o sucesso do envio do RabbitMQ".

## Decisao

Criar um logger proprio usando apenas codigos ANSI nativos, sem nenhuma
dependencia externa (zerolog, logrus, etc.).

## Motivos

1. **Zero dependencias:** menos complexidade, menos vulnerabilidades
2. **Simples e funcional:** o necessario para o projeto — niveis, cores, icones
3. **Codigo pequeno:** ~70 linhas, facil de manter
4. **Funciona em qualquer terminal moderno:** PowerShell, Windows Terminal,
   VS Code, Git Bash, Linux

## Implementacao

```go
const (
    corResetar  = "\033[0m"
    corAzul     = "\033[1;34m"
    corVerde    = "\033[1;32m"
    corAmarelo  = "\033[1;33m"
    corVermelho = "\033[1;31m"
    corCiano    = "\033[1;36m"
    corMagenta  = "\033[1;35m"
)
```

| Funcao | Cor | Icone | Uso |
|---|---|---|---|
| `Info()` | Azul | ℹ | Mensagens informativas |
| `Sucesso()` | Verde | ✔ | Operacoes concluidas |
| `Aviso()` | Amarelo | ⚠ | Retry/erros temporarios |
| `Erro()` | Vermelho | ✘ | Falhas criticas |
| `Destaque()` | Magenta | ℹ | Eventos importantes |
| `Inicio()` | Magenta | ▶ | Inicio de execucao |

## Impacto

- Substituiu todos os `log.Println/Printf` do projeto
- Logs mais legiveis com timestamp + tag + nivel + mensagem
- Facil de estender (novos niveis, formatos)

## Arquivos envolvidos
- `internal/infra/logger/logger.go` — implementacao (72 linhas)
- Todos os arquivos que usavam `log` foram atualizados
