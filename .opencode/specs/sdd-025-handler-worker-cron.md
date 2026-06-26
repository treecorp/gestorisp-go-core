# SDD-025 — Handler Worker + Cron

**Status:** Pendente
**Autor:** Knowledge Engineer
**Prioridade:** Alta
**Dependencias:** SDD-018 (`entity`), SDD-020 (`repositorio/pagamento`), SDD-021 (`repositorio/infra`), SDD-022 (`service/pagamento`), SDD-023 (`service/bloqueio`)

## 1. Objetivo

Mover `internal/worker/` e `internal/cron/` para `internal/handler/`, atualizando imports para utilizar os novos pacotes de repositório e service. Manter a estrutura de handlers original (mesmo arquivo, mesmas funções), apenas atualizando chamadas internas.

## 2. Escopo

### 2.1 internal/handler/worker/

Mover todo o conteúdo de `internal/worker/` para `internal/handler/worker/`, mantendo a mesma estrutura de arquivos:

```
internal/handler/worker/
  main.go              <- Handler principal do worker
  processar.go         <- Lógica de processamento (agora usando service)
  desconexao.go        <- Handlers de desconexão
```

Mudanças principais:
- Substituir chamadas diretas a DB por chamadas a `repositorio.*`
- Substituir lógica de negócio inline por chamadas a `service.pagamento.*` e `service.bloqueio.*`
- Manter a assinatura das funções handlers (recebem `http.ResponseWriter`, `*http.Request`)

### 2.2 internal/handler/cron/

Mover todo o conteúdo de `internal/cron/` para `internal/handler/cron/`:

```
internal/handler/cron/
  agendador.go         <- Scheduler (mantido, apenas atualiza imports)
  tarefas/
    base.go            <- Tarefas agora usam services + repositorios
    bloqueio.go        <- Tarefa de bloqueio (service.bloqueio)
    pagamento.go       <- Tarefa de pagamento (service.pagamento)
    sincronia.go       <- Tarefa de sincronia (repositorio)
    radacct.go         <- Tarefa radacct (repositorio.radacct)
    cluster.go         <- Tarefa cluster (repositorio.cluster)
```

### 2.3 Atualizações de import

Substituições de import necessárias:

| Import antigo | Novo import |
|--------------|-------------|
| `internal/cron/` | `internal/handler/cron/` |
| `internal/worker/` | `internal/handler/worker/` |
| `internal/dominio/` | `internal/entity/` |
| `internal/pagamento/` | `internal/service/pagamento/` + `internal/repositorio/` |
| `internal/gateway/` | `internal/handler/gateway/` |
| `internal/api/` | `internal/handler/api/` |

## 3. Estrutura de diretórios resultante

```
internal/handler/
  gateway/             <- (ex-internal/gateway)
    server.go
    auth.go
    webhook.go
    doc.go
  api/                 <- (ex-internal/api)
    server.go
    webhook_iugu.go
    routeros.go
    swagger.go
    errors.go
    doc.go
  worker/              <- (ex-internal/worker)
    main.go
    processar.go
    desconexao.go
    doc.go
  cron/                <- (ex-internal/cron)
    agendador.go
    tarefas/
      base.go
      bloqueio.go
      pagamento.go
      sincronia.go
      radacct.go
      cluster.go
    doc.go
```

## 4. Arquivos a criar

| Arquivo | Ação |
|---------|------|
| `internal/handler/worker/main.go` | Criar — mover de `internal/worker/` |
| `internal/handler/worker/processar.go` | Criar — mover de `internal/worker/` |
| `internal/handler/worker/desconexao.go` | Criar — mover de `internal/worker/` |
| `internal/handler/worker/doc.go` | Criar — package-level doc |
| `internal/handler/cron/agendador.go` | Criar — mover de `internal/cron/` |
| `internal/handler/cron/tarefas/base.go` | Criar — mover de `internal/cron/tarefas/` |
| `internal/handler/cron/tarefas/bloqueio.go` | Criar — mover de `internal/cron/tarefas/` |
| `internal/handler/cron/tarefas/pagamento.go` | Criar — mover de `internal/cron/tarefas/` |
| `internal/handler/cron/tarefas/sincronia.go` | Criar — mover de `internal/cron/tarefas/` |
| `internal/handler/cron/tarefas/radacct.go` | Criar — mover de `internal/cron/tarefas/` |
| `internal/handler/cron/tarefas/cluster.go` | Criar — mover de `internal/cron/tarefas/` |
| `internal/handler/cron/doc.go` | Criar — package-level doc |

## 5. Documentação

Toda função handler deve ter **doc comment em português**:

```go
// Package worker implementa os handlers HTTP para o worker de
// processamento de filas, incluindo desconexão de contratos e
// sincronização de pagamentos.
package worker
```

```go
// Package cron implementa o agendador de tarefas periódicas e os
// handlers para cada job (bloqueio, pagamento, sincronia, radacct,
// cluster).
package cron
```

## 6. Critérios de Aceite

- [ ] `internal/handler/worker/` compila sem erros
- [ ] `internal/handler/cron/` compila sem erros
- [ ] Todos os imports atualizados para nova estrutura
- [ ] Comportamento dos jobs cron idêntico ao original
- [ ] Comportamento dos workers idêntico ao original
- [ ] `cmd/gestor/main.go` atualizado para importar novos caminhos
- [ ] Todas as funções possuem doc comment em português
- [ ] `doc.go` presente em todos os pacotes
