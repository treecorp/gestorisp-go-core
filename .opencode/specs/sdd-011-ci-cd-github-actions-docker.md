# SDD-011 — CI/CD com GitHub Actions + Docker unificado

**Status:** Aprovado
**Autor:** Dev Backend
**Prioridade:** Alta
**Dependencias:** Secrets `GH_TOKEN` e `GH_USER` configurados no repositorio GitHub

## 1. Objetivo
Automatizar build, verificacao e publicacao das imagens Docker do projeto sempre que houver push na `main` ou PR.

## 2. Dockerfile Unificado

### Mudanca principal
Um unico `Dockerfile` compila **ambos os binarios** (`gestor` e `worker`) e decide qual executar via variavel de ambiente `SERVICO`.

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gestor ./cmd/gestor && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /worker ./cmd/worker

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata

ENV TZ=America/Sao_Paulo \
    SERVICO=cron

WORKDIR /app
COPY --from=build /gestor .
COPY --from=build /worker .

CMD ["sh", "-c", "if [ \"$SERVICO\" = \"worker\" ]; then exec ./worker; else exec ./gestor; fi"]
```

### ENV `SERVICO`

| Valor | Binario executado | Uso |
|-------|-------------------|-----|
| `cron` (padrao) | `./gestor` | Agendador de tarefas |
| `worker` | `./worker` | Consumidor RabbitMQ |

## 3. Workflow CI

**Arquivo:** `.github/workflows/ci.yml`

### Eventos
- `push` na branch `main`
- `pull_request` para `main`

### Etapas

| Etapa | Acao |
|-------|------|
| 1 | `actions/checkout@v4` |
| 2 | `actions/setup-go@v5` com Go 1.22 |
| 3 | `go vet ./...` |
| 4 | `go build ./...` |
| 5 | `docker build -t ghcr.io/$GH_USER/gestor:latest` |
| 6 | Se `push` na `main`: login GHCR + `docker push` |

### Imagem publicada
```
ghcr.io/<GH_USER>/gestor:latest
```

### Secrets necessarios

| Secret | Descricao |
|--------|-----------|
| `GH_TOKEN` | Token de acesso do GitHub com permissao `write:packages` |
| `GH_USER` | Nome de usuario GitHub (owner do repositorio) |

## 4. Arquivos afetados

| Arquivo | Acao | Descricao |
|---------|------|-----------|
| `Dockerfile` | **Substituir** | Compila 2 binarios, entrypoint por `SERVICO` |
| `.github/workflows/ci.yml` | **Criar** | Workflow CI/CD |

## 5. Exemplo de uso

```bash
# Cron (padrao)
docker run ghcr.io/meuusuario/gestor:latest

# Worker
docker run -e SERVICO=worker ghcr.io/meuusuario/gestor:latest
```
