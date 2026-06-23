
# Estrutura do Projeto

## Arvore de Diretorios

```
gestor/
│
├── cmd/                                # Pontos de entrada do sistema
│   └── gestor/
│       └── main.go                     # Entry point: inicia tudo
│
├── internal/                           # Codigo interno (nao exportavel)
│   │
│   ├── config/                         # Configuracoes centralizadas
│   │   └── config.go                   #   - Le variaveis de ambiente
│   │                                   #   - Define structs Config, BancoConfig, RabbitMQConfig
│   │                                   #   - Valores padrao hardcoded (temporario)
│   │
│   ├── dominio/                        # Entidades de negocio (modelos)
│   │   └── instancia.go                #   - Struct Instancia (id, token, credenciais DB)
│   │                                   #   - Metodo GetID()
│   │
│   ├── infra/                          # Infraestrutura compartilhada
│   │   │
│   │   ├── banco/                      # Banco de dados MySQL
│   │   │   ├── mysql.go               #   - Pool de conexoes (10 max)
│   │   │   │                           #   - Conectar(), ConectarComRetry() (loop infinito)
│   │   │   │                           #   - Ping() com reconexao automatica
│   │   │   │                           #   - Fechar()
│   │   │   └── gispadm.go             #   - BuscarInstanciasAtivas() (query GISPADM)
│   │   │
│   │   ├── mensageria/                 # Mensageria RabbitMQ
│   │   │   └── rabbit.go              #   - Conexao com reconexao automatica
│   │   │                               #   - NotifyClose listener
│   │   │                               #   - PublicarInstancia() (JSON → Base64 → fila)
│   │   │                               #   - ConectarComRetry() (loop infinito)
│   │   │
│   │   └── logger/                     # Logger colorido
│   │       └── logger.go              #   - Info(), Sucesso(), Aviso(), Erro()
│   │                                   #   - Destaque(), Inicio()
│   │                                   #   - Cores ANSI + icones
│   │
│   ├── cron/                           # Agendador de tarefas
│   │   ├── agendador.go               #   - Scheduler robfig/cron
│   │   │                               #   - Registro de tarefas (config-driven)
│   │   │                               #   - Iniciar(), Parar()
│   │   └── tarefas/
│   │       └── base.go                #   - ExecutarParaTodasInstancias()
│   │                                   #   - publicarComRetry() (3 tentativas)
│   │
│   ├── worker/                         # 🔄 FASE 2 - Workers RabbitMQ
│   │   └── .placeholder               #   Consumidores que processam as filas
│   │
│   └── api/                            # 📅 FASE 3 - API HTTP
│       └── .placeholder               #   Gateway de pagamentos, REST API
│
├── Dockerfile                          # Build multi-stage
├── go.mod                              # Modulo Go
├── go.sum                              # Checksum das dependencias
├── .env.exemplo                        # Template de variaveis de ambiente
├── README.md                           # Documentacao principal
│
└── docs/                               # Documentacao detalhada
    ├── ARQUITETURA.md                  # Arquitetura e decisoes tecnicas
    ├── ESTRUTURA.md                    # Este arquivo
    ├── FUNCIONALIDADES.md              # Descricao das tarefas cron
    └── MIGRACAO.md                     # Plano de migracao e roadmap
```

## Pacotes e Responsabilidades

### `cmd/gestor/main.go`

Ponto de entrada do sistema. Executa em ordem:

1. **Config**: carrega variaveis de ambiente
2. **Banco**: conecta no MySQL GISPADM (loop infinito ate conseguir)
3. **RabbitMQ**: conecta no RabbitMQ (loop infinito ate conseguir)
4. **Agendador**: registra as 7 tarefas cron e inicia
5. **Graceful shutdown**: aguarda SIGINT/SIGTERM e encerra de forma controlada

### `internal/config/config.go`

Centraliza todas as configuracoes do sistema. Le de variaveis de ambiente com fallback para valores hardcoded (temporario ate a migracao completa).

```go
type Config struct {
    Banco    BancoConfig
    RabbitMQ RabbitMQConfig
}
```

### `internal/dominio/instancia.go`

Entidade principal do dominio. Representa uma instancia ativa do GISP registrada no banco central.

```go
type Instancia struct {
    ID        int    // Identificador unico
    Token     string // Token de autenticacao
    EnvDBHost string // Host do banco da instancia
    EnvDBUser string // Usuario do banco da instancia
    EnvDBPass string // Senha do banco da instancia
    EnvDBName string // Nome do banco da instancia
}
```

### `internal/infra/banco/mysql.go`

Gerencia o pool de conexoes MySQL. Caracteristicas:

- **Pool**: 10 conexoes max, 5 idle, 5 minutos de vida
- **ConectarComRetry**: loop infinito com backoff (2s → 4s → 8s → ... → 60s)
- **Ping**: verifica saude da conexao; se falhar, reconecta

### `internal/infra/banco/gispadm.go`

Contem as queries especificas do GISPADM. Atualmente:

```sql
SELECT id, token, env_dbname, env_dbuser, env_dbpass, env_dbhost
FROM instancias
WHERE app = 'GISP-FULL' AND status = 'Ativo'
```

### `internal/infra/mensageria/rabbit.go`

Gerencia a conexao com RabbitMQ. Caracteristicas:

- **ConectarComRetry**: loop infinito com backoff
- **NotifyClose**: goroutine que escuta fechamento da conexao e reconecta
- **PublicarInstancia**: serializa Instancia → JSON → Base64 → publica na fila
- **Filas non-durable**: mesmo comportamento do server.js original

### `internal/infra/logger/logger.go`

Logger colorido sem dependencias externas. Usa codigos ANSI.

| Funcao | Cor | Icone | Uso |
|---|---|---|---|
| `Info()` | Azul | ℹ | Mensagens informativas |
| `Sucesso()` | Verde | ✔ | Operacoes concluidas |
| `Aviso()` | Amarelo | ⚠ | Retry/erros temporarios |
| `Erro()` | Vermelho | ✘ | Falhas criticas |
| `Destaque()` | Magenta | ℹ | Eventos importantes |
| `Inicio()` | Magenta | ▶ | Inicio de execucao |

### `internal/cron/agendador.go`

Agendador de tarefas usando `robfig/cron/v3`. Caracteristicas:

- **Config-driven**: tarefas definidas como slices de `TarefaRegistro`
- **Segundos**: configurado com `cron.WithSeconds()` para precisao
- **7 tarefas**: registradas no `Iniciar()`

### `internal/cron/tarefas/base.go`

Logica comum a todas as tarefas. Cada tarefa executa:

1. Busca instancias ativas no GISPADM
2. Para cada instancia, publica na fila RabbitMQ
3. Se falhar, tenta 3 vezes (1s, 2s, 4s)
4. Loga o resultado

## Convencoes

### Nomenclatura

- **Pacotes**: em portugues, minusculo, sem underlines
- **Arquivos**: em portugues, minusculo, com underlines
- **Funcoes**: PascalCase (exportadas) ou camelCase (privadas)
- **Variaveis**: camelCase
- **Constantes**: PascalCase

### Tratamento de Erros

- Erros sao sempre propagados com `fmt.Errorf("contexto: %w", err)`
- Funcoes de infraestrutura retornam `error`
- Logging e feito pelo pacote `logger`, nunca por `fmt.Print`
- Erros em lote (publicacao de multiplas instancias) nao interrompem o lote

### Dependencias Externas

| Dependencia | Versao | Justificativa |
|---|---|---|
| `github.com/robfig/cron/v3` | v3.0.1 | Agendador cron maduro e testado |
| `github.com/streadway/amqp` | v1.1.0 | Cliente RabbitMQ oficial |
| `github.com/go-sql-driver/mysql` | v1.9.3 | Driver MySQL padrao |
| `filippo.io/edwards25519` | v1.1.0 | Dependencia do driver MySQL |
