# SDD-012 — Ajuste dos agendamentos cron

**Status:** Aprovado
**Autor:** Dev Backend
**Prioridade:** Media

## 1. Objetivo
Ajustar as expressoes cron de 4 tarefas para refletir as novas frequencias.

## 2. Tabela de alteracoes

| Tarefa | Fila | Expressao atual | Nova expressao | Frequencia |
|--------|------|----------------|----------------|------------|
| `cron_um` | `cron_1` | `0 */5 0,3-23 * * *` | `0 */1 * * * *` | A cada **1 minuto**, sem restricao de horario |
| `executar_cluster` | `run_cluster` | `*/30 * * * * *` | `0 */1 * * * *` | A cada **1 minuto** |
| `sincronizar_conexoes` | `sync_conexoes_radius_arquivo` | `*/30 * * * * *` | `0 */1 * * * *` | A cada **1 minuto** |
| `verificar_status_pop` | `check_pop_status` | `*/30 * * * * *` | *(mantido)* | A cada **30 segundos** |

### Tarefas nao alteradas

| Tarefa | Expressao | Frequencia |
|--------|-----------|------------|
| `reparar_radius` | `0 30 0 * * *` | Diario 00:30 |
| `limpeza_logs` | `0 30 0 * * *` | Diario 00:30 |
| `listar_clientes_vencidos` | `0 10 14 * * *` | Diario 14:10 |

## 3. Arquivo modificado

**`cmd/gestor/main.go`** — linhas 37-40

## 4. Build

```bash
go vet ./cmd/gestor && go build ./...
```
