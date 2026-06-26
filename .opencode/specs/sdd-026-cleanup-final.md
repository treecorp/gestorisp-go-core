# SDD-026 — Cleanup Final

**Status:** Pendente
**Autor:** Knowledge Engineer
**Prioridade:** Alta
**Dependencias:** SDD-018, SDD-019, SDD-020, SDD-021, SDD-022, SDD-023, SDD-024, SDD-025

## 1. Objetivo

Remover pacotes antigos após a migração completa, atualizar entrypoints em `cmd/`, verificar build e documentar a nova arquitetura no Banco de Memória do Projeto (BMP).

## 2. Escopo

### 2.1 Remoção de pacotes antigos

Após confirmar que nenhum import restante referencia esses pacotes:

| Pacote | Ação | Conteúdo movido para |
|--------|------|----------------------|
| `internal/pagamento/` | Remover | `service/pagamento/` + `repositorio/` + `helpers/` + `lib/iugu/` |
| `internal/dominio/` | Remover | `entity/` |
| `internal/gateway/` | Remover | `handler/gateway/` |
| `internal/api/` | Remover | `handler/api/` |
| `internal/worker/` | Remover | `handler/worker/` |
| `internal/cron/` | Remover | `handler/cron/` |

### 2.2 Atualização de entrypoints

Atualizar `cmd/` entrypoints para refletir a nova estrutura:

| Entrypoint | Alterações |
|-----------|------------|
| `cmd/gestor/main.go` | Atualizar imports: `handler/cron` + `handler/worker` |
| `cmd/gateway/main.go` | Atualizar imports: `handler/gateway` |
| `cmd/api/main.go` | Atualizar imports: `handler/api` |
| `cmd/worker/main.go` | Atualizar imports: `handler/worker` |

### 2.3 Atualização de internal/infra/banco/gispinstancia.go

O arquivo `internal/infra/banco/gispinstancia.go` tinha queries de POP que foram extraídas para `repositorio/pop_repo.go`. Remover as funções duplicadas e atualizar os imports nos arquivos que ainda usam `gispinstancia.go`.

### 2.4 Verificação de build

```bash
go build ./...
go vet ./...
```

Corrigir qualquer erro de compilação ou vet até que ambos passem limpos.

### 2.5 Dockerfile

Revisar `Dockerfile` para garantir que os caminhos de build estão corretos:

```dockerfile
# Antigo
COPY cmd/gestor ./cmd/gestor
COPY internal/ ./internal/

# Novo (verificar se paths continuam válidos)
COPY cmd/ ./cmd/
COPY internal/ ./internal/
```

### 2.6 Documentação no BMP

#### Nova entrada em memory/decisions/

Arquivo: `.opencode/memory/decisions/MEM-XXX.md`

```yaml
---
id: MEM-XXX
type: decision
tags: [arquitetura, refatoracao, estrutura]
date: 2026-06-26
related_files:
  - internal/entity/
  - internal/helpers/
  - internal/lib/iugu/
  - internal/repositorio/
  - internal/service/pagamento/
  - internal/service/bloqueio/
  - internal/handler/
---
```

Conteúdo: Descrever a decisão de migrar de pacotes planos para uma arquitetura em camadas (entity → repositorio → service → handler).

#### Atualização de memory/index.md

- Remover entradas antigas (se referiam a pacotes removidos)
- Adicionar entradas para novos pacotes
- Atualizar estatísticas

## 3. Ordem de execução

1. Remover imports antigos de `cmd/` entrypoints
2. Remover pacotes antigos um a um, compilando após cada remoção
3. Atualizar `internal/infra/banco/gispinstancia.go`
4. `go build ./...` e `go vet ./...`
5. Revisar `Dockerfile`
6. Criar decisão no BMP
7. Atualizar `memory/index.md`

## 4. Arquivos a modificar

| Arquivo | Ação |
|---------|------|
| `cmd/gestor/main.go` | Alterar — imports |
| `cmd/gateway/main.go` | Alterar — imports |
| `cmd/api/main.go` | Alterar — imports |
| `cmd/worker/main.go` | Alterar — imports |
| `internal/infra/banco/gispinstancia.go` | Alterar — remover funções movidas |
| `Dockerfile` | Revisar — paths de build |
| `.opencode/memory/decisions/MEM-XXX.md` | Criar — decisão arquitetural |
| `.opencode/memory/index.md` | Alterar — estatísticas + índice |

## 5. Documentação

Todas as funções modificadas devem ter **doc comment em português** atualizado.

## 6. Critérios de Aceite

- [ ] `go build ./...` passa sem erros
- [ ] `go vet ./...` passa sem erros
- [ ] Nenhum pacote antigo (`pagamento/`, `dominio/`, `gateway/`, `api/`, `worker/`, `cron/`) permanece no código
- [ ] `cmd/` entrypoints compilam e funcionam
- [ ] `Dockerfile` produz imagem funcional
- [ ] Decisão arquitetural registrada no BMP
- [ ] `memory/index.md` atualizado
