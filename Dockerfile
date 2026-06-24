FROM golang:1.26-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gestor ./cmd/gestor && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /worker ./cmd/worker && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gateway ./cmd/gateway

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

ENV TZ=America/Sao_Paulo \
    SERVICO=cron

WORKDIR /app
COPY --from=build /gestor .
COPY --from=build /worker .
COPY --from=build /gateway .

CMD ["sh", "-c", "case \"${SERVICO}\" in worker) exec ./worker ;; gateway) exec ./gateway ;; *) exec ./gestor ;; esac"]
