
# 🧠 Banco de Memoria do Projeto (BMP) — gestorisp-go-core

**Status:** ✅ Ativo (Fase 1 - Cron completo, Fase 2 - Gateway Iugu assincrono)

**Ultima atualizacao:** 25/06/2026 (16:30)

## Estatisticas

| Categoria | Quantidade |
|---|---|
| Decisoes | 12 |
| Bugs | 2 |
| Convencoes | 4 |
| Gotchas | 9 |
| Padroes | 2 |
| Bugs | 2 |
| Convencoes | 4 |
| Gotchas | 9 |
| Padroes | 2 |
| Specs | 10 |
| **Total** | **39** |

## Indice

### Decisoes Tecnicas
- [001 - Go publica direto no RabbitMQ](decisions/001-go-publica-direto-rabbitmq.md)
- [002 - Retry infinito com backoff](decisions/002-retry-infinito-backoff.md)
- [003 - Logger com ANSI puro sem dependencias](decisions/003-logger-ansi-sem-dependencias.md)
- [004 - Estrutura modular para comportar fases futuras](decisions/004-estrutura-modular-fases.md)
- [005 - Adocao do Banco de Memoria do Projeto (BMP)](decisions/005-adocao-bmp-memoria.md)
- [006 - Port do cron_1 com go-routeros + fuso centralizado](decisions/006-cron-1-routeros-fuso.md)
- [007 - Abordagem mista para queries (SELECT batch JOIN + UPDATE individual)](decisions/007-batch-join-misto.md)
- [008 - Precedencia de dias_bloqueio + permitir_bloqueio no bloqueio de inadimplentes](decisions/008-dias-bloqueio-permitir-bloqueio.md)

### Decisoes (Fase 2 - Gateway)
- [009 - Gateway webhook Iugu como binario standalone HTTP](decisions/009-gateway-iugu-standalone-http.md)
- [010 - Gateway assincrono via RabbitMQ + Worker Desconexao](decisions/010-gateway-assincrono-worker-desconexao.md)

### Decisoes (Fase 3 - SO / Melhorias)
- [011 - listar_clientes_vencidos publica na fila desconectar_contrato em vez de chamar RouterOS direto](decisions/011-listar-clientes-vencidos-desconexao-fila.md)

### Bugs
- [001 - Nome do banco incorreto: gispadm vs gisp_adm](bugs/001-nome-banco-incorreto.md)
- [002 - CAST dias_bloqueio retorna 0 para string vazia](bugs/002-cast-dias-bloqueio-string-vazia.md)

### Convencoes
- [001 - Codigo e comentarios em portugues](conventions/001-codigo-em-portugues.md)
- [002 - Erros com fmt.Errorf](conventions/002-erros-com-fmt-errorf.md)
- [003 - Pacotes sem underline](conventions/003-pacotes-sem-underline.md)
- [004 - Conexao unica compartilhada](conventions/004-conexao-unica-compartilhada.md)

### Gotchas (licoes aprendidas)
- [001 - Porta RabbitMQ nao padrao 31837](gotchas/001-porta-rabbitmq-nao-padrao.md)
- [002 - Type assertion em interface no retry](gotchas/002-type-assertion-retry.md)
- [003 - WS2 retorna JSON array, nao objeto](gotchas/003-ws2-retorna-array-json.md)
- [004 - Worker deve usar prefetch 1 (uma mensagem por vez)](gotchas/004-prefetch-1-worker.md)
- [005 - POP pode ficar offline entre check_pop_status e cron_1](gotchas/005-pop-status-stale.md)
- [006 - contrato_pop_id pode ser NULL na radacct](gotchas/006-contrato-pop-id-null.md)
- [007 - Colunas IPv6 ausentes em radacct quebram SELECT fixo](gotchas/007-colunas-ipv6-ausentes-radacct.md)
- [008 - CAST('' AS UNSIGNED) retorna 0, nao NULL](gotchas/008-cast-string-vazia-unsigned.md)
- [009 - PowerShell quoting com curl.exe para JSON quebra payload](gotchas/009-powershell-quoting-curl-json.md)

### Padroes
- [001 - Tarefa cron config-driven](patterns/001-tarefa-cron-config-driven.md)
- [002 - Retry com backoff exponencial](patterns/002-retry-backoff-exponencial.md)

---
## Progresso

### Migrados (7/7)
| Worker | Handler | Cron |
|--------|---------|------|
| `cron_1` | `cron_1.go` (5 sub-rotinas) | `0 */5 0,3-23 * * *` |
| `run_cluster` | `run_cluster.go` | `*/30 * * * * *` |
| `check_pop_status` | `check_pop_status.go` | `*/30 * * * * *` |
| `sync_conexoes_radius_arquivo` | `sync_conexoes_radius_arquivo.go` | `*/30 * * * * *` |
| `repair_radius_acctstoptime` | `repair_radius_acctstoptime.go` | `0 30 0 * * *` |
| `limpeza_logs` | `limpeza_logs.go` | `0 30 0 * * *` |
| `listar_clientes_vencidos` | `listar_clientes_vencidos.go` | `0 10 14 * * *` |

### Melhorias recentes
- `listar_clientes_vencidos` agora publica na fila `desconectar_contrato` em vez de chamar RouterOS direto (desacoplamento + retry infinito)
- `dias_bloqueio` varchar: parse manual em Go com `strings.TrimSpace` + `strconv.Atoi` (HOTFIX-004)
- Testes unitarios do gateway Iugu: 16 testes com dados reais (3 faturas reais extraidas do banco)
- `formatarMoeda` corrigido: padding de zeros para centavos (`"5"` → `"0,05"`)
- Gateway aceita JSON puro (`Content-Type: application/json`) — parse correto de event + data
- Testado em producao com 3 webhooks reais (form + JSON) — todos processados com sucesso
- HOTFIX-005: gateway filtra apenas `invoice.status_changed` — eventos `invoice.created`/`invoice.released` ignorados (igual PHP legado)

### Pendentes
- Reativar cron `listar_clientes_vencidos` em `cmd/gestor/main.go`

### Fase 2 — Gateway Iugu (portado, agora assincrono)
| Binario | Porta | Rota |
|---------|-------|------|
| `cmd/gateway` | 8082 | `POST /pagamentos/iugu/gatilho/{token}` |

### Workers (novos)
| Worker | Fila | Retry | Descricao |
|--------|------|-------|-----------|
| `processar_pagamento_iugu` | `processar_pagamento_iugu` | Max 5 | Processa baixa com TX MySQL |
| `desconectar_contrato` | `desconectar_contrato` | Infinito | Desconecta cliente do RouterOS |

### Especificacoes
- [SDD-008 - repair_radius_acctstoptime](../specs/sdd-008-repair-radius-acctstoptime.md)
- [SDD-009 - limpeza_logs](../specs/sdd-009-limpeza-logs.md)
- [SDD-010 - listar_clientes_vencidos](../specs/sdd-010-listar-clientes-vencidos.md)
- [SDD-011 - CI/CD GitHub Actions + Docker](../specs/sdd-011-ci-cd-github-actions-docker.md)
- [SDD-012 - Ajuste agendamentos cron](../specs/sdd-012-ajuste-agendamentos-cron.md)
- [SDD-014 - Gateway pagamentos Iugu](../specs/sdd-014-gateway-pagamentos-iugu.md)
- [SDD-015 - Gateway assincrono + Worker Desconexao](../specs/sdd-015-gateway-assincrono-worker-desconexao.md)
- [SDD-016 - Testes unitarios do Gateway Iugu](../specs/sdd-016-testes-unitarios-gateway-iugu.md)
- [HOTFIX-004 - Parse dias_bloqueio varchar com fallup](../specs/hotfix-004-cast-dias-bloqueio-varchar.md)
- [HOTFIX-005 - Filtrar eventos Iugu no gateway (apenas invoice.status_changed)](../specs/hotfix-005-filtrar-eventos-gateway.md)

---
> **Como usar:** sempre consulte as categorias relevantes antes de comecar uma tarefa.
> Ao finalizar, registre novos aprendizados e atualize o indice.
