# =============================================================================
# Etapa 1: compilacao do binario Go
# =============================================================================
FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gestor ./cmd/gestor

# =============================================================================
# Etapa 2: imagem final minima
# =============================================================================
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

# ─────────────────────────────────────────────────────────────────────────────
# ATENCAO: Variaveis hardcoded temporariamente
# TODO: Remover quando o backend Go estiver completo e migrar para um
#       provedor de configuracao externo (Vault, Docker Secrets, etc.)
# ─────────────────────────────────────────────────────────────────────────────
ENV DB_GISPADM_HOST=177.136.249.51 \
    DB_GISPADM_PORT=31034 \
    DB_GISPADM_USER=gestorisp \
    DB_GISPADM_PASS="WM33223200kl**" \
    DB_GISPADM_DBNAME=gisp_adm \
    RABBITMQ_HOST=172.16.12.10 \
    RABBITMQ_PORT=31837 \
    RABBITMQ_USER=guest \
    RABBITMQ_PASS=guest
# ─────────────────────────────────────────────────────────────────────────────

WORKDIR /app
COPY --from=build /gestor .

CMD ["./gestor"]
