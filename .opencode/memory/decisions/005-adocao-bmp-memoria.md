
# Decisao 005: Adocao do Banco de Memoria do Projeto (BMP)

**Data:** 23/06/2026
**Autor:** opencode
**Status:** ✅ Implementado

## Contexto

O projeto possui muitas decisoes tecnicas, convencoes, bugs corrigidos e
padroes que precisam ser documentados para consulta futura. Sem um registro
central, o agente opencode corre o risco de:
- Repetir decisoes ja tomadas
- Ignorar bugs conhecidos
- Quebrar convencoes estabelecidas
- Perder contexto entre sessoes

## Decisao

Adotar o padrao **Banco de Memoria do Projeto (BMP)** com estrutura
categorizada em `.opencode/memory/`.

## Estrutura

```
.opencode/memory/
  index.md         <- Indice central com estatisticas
  decisions/       <- Decisoes tecnicas importantes
  bugs/            <- Bugs encontrados e corrigidos
  conventions/     <- Convencoes de codigo
  gotchas/         <- Licoes aprendidas (armadilhas)
  patterns/        <- Padroes de codigo recorrentes
  archived/        <- Entradas antigas (historico)
```

## Obrigatoriedade

O `opencode.json` possui `instructions: [".opencode/AGENTS.md"]` que faz o
AGENTS.md ser carregado em toda sessao. O AGENTS.md exige:

**Antes de qualquer acao:** consultar a memoria
**Depois de qualquer acao:** atualizar a memoria

Nao ha como ignorar — esta no prompt base do agente.

## Motivos

1. **Contexto persistente:** o agente nao perde o que foi feito entre sessoes
2. **Rastreabilidade:** toda decisao fica registrada com data e motivo
3. **Onboarding:** novo desenvolvedor le o index.md e entende o projeto
4. **Disciplina:** o AGENTS.md obriga a consulta e atualizacao

## Impacto

- Criado `.opencode/opencode.json` com `instructions`
- Criado `.opencode/AGENTS.md` com regras obrigatorias
- Criados 14 arquivos de memoria preenchidos desde o inicio do projeto
