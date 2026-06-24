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
