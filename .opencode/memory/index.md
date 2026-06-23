
# 🧠 Banco de Memoria do Projeto (BMP) — gestorisp-go-core

**Status:** ✅ Ativo (Fase 1 - Cron)

**Ultima atualizacao:** 23/06/2026

## Estatisticas

| Categoria | Quantidade |
|---|---|
| Decisoes | 5 |
| Bugs | 1 |
| Convencoes | 4 |
| Gotchas | 2 |
| Padroes | 2 |
| **Total** | **14** |

## Indice

### Decisoes Tecnicas
- [001 - Go publica direto no RabbitMQ](decisions/001-go-publica-direto-rabbitmq.md)
- [002 - Retry infinito com backoff](decisions/002-retry-infinito-backoff.md)
- [003 - Logger com ANSI puro sem dependencias](decisions/003-logger-ansi-sem-dependencias.md)
- [004 - Estrutura modular para comportar fases futuras](decisions/004-estrutura-modular-fases.md)
- [005 - Adocao do Banco de Memoria do Projeto (BMP)](decisions/005-adocao-bmp-memoria.md)

### Bugs
- [001 - Nome do banco incorreto: gispadm vs gisp_adm](bugs/001-nome-banco-incorreto.md)

### Convencoes
- [001 - Codigo e comentarios em portugues](conventions/001-codigo-em-portugues.md)
- [002 - Erros com fmt.Errorf](conventions/002-erros-com-fmt-errorf.md)
- [003 - Pacotes sem underline](conventions/003-pacotes-sem-underline.md)
- [004 - Conexao unica compartilhada](conventions/004-conexao-unica-compartilhada.md)

### Gotchas (licoes aprendidas)
- [001 - Porta RabbitMQ nao padrao 31837](gotchas/001-porta-rabbitmq-nao-padrao.md)
- [002 - Type assertion em interface no retry](gotchas/002-type-assertion-retry.md)

### Padroes
- [001 - Tarefa cron config-driven](patterns/001-tarefa-cron-config-driven.md)
- [002 - Retry com backoff exponencial](patterns/002-retry-backoff-exponencial.md)

---
> **Como usar:** sempre consulte as categorias relevantes antes de comecar uma tarefa.
> Ao finalizar, registre novos aprendizados e atualize o indice.
