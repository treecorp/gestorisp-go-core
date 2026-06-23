
# Instrucoes Obrigatorias — gestorisp-go-core

## 🧠 MEMORIA DO PROJETO — LEIA ANTES DE QUALQUER ACAO
Este projeto mantem um Banco de Memoria em `.opencode/memory/`.
O uso da memoria e **OBRIGATORIO** em todas as sessoes.

### Antes de qualquer tarefa:
1. Leia `memory/index.md` para contexto geral e historico
2. Consulte `memory/decisions/` para nao repetir decisoes passadas
3. Consulte `memory/conventions/` para seguir padroes do projeto
4. Consulte `memory/gotchas/` para evitar erros ja conhecidos
5. Consulte `memory/bugs/` para conhecer bugs anteriores
6. Consulte `memory/patterns/` para reutilizar padroes existentes

### Ao finalizar qualquer tarefa:
1. Crie entrada em `memory/decisions/` se nova decisao tecnica
2. Crie entrada em `memory/bugs/` se novo bug encontrado
3. Crie entrada em `memory/gotchas/` se novo aprendizado
4. Crie entrada em `memory/patterns/` se novo padrao identificado
5. **ATUALIZE** `memory/index.md` — estatisticas + indice

## ⚙️ Comandos do Projeto
- Build: `go build ./...`
- Vet: `go vet ./...`
- Run: `go run ./cmd/gestor`

## 📐 Convencoes de Codigo
- Codigo e comentarios em **portugues** (obrigatorio)
- Nomes de pacotes sem underlines
- Erros com `fmt.Errorf("contexto: %w", err)`
- Usar `logger.Info/Sucesso/Aviso/Erro`, nunca `fmt.Print` ou `log.Println`
- Sistema **NUNCA** pode cair — usar retry infinito, nunca `os.Exit`
- Conexoes com backoff exponencial (2s -> 4s -> 8s -> ... -> 60s)

## 📂 Estrutura do Projeto
```
cmd/gestor/main.go              <- Entry point
internal/
  config/config.go              <- Env vars
  dominio/instancia.go          <- Entidades
  infra/
    banco/mysql.go              <- Pool MySQL
    banco/gispadm.go            <- Query GISPADM
    mensageria/rabbit.go        <- Publisher RabbitMQ
    logger/logger.go            <- Logger colorido ANSI
  cron/
    agendador.go                <- Scheduler
    tarefas/base.go             <- Logica dos jobs
  worker/                       <- (Fase 2)
  api/                          <- (Fase 3)
docs/                           <- Documentacao
.opencode/                      <- Config + memoria
```
