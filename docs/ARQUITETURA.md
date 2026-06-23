
# Arquitetura do Sistema

## Visao Geral

O Gestor ISP e um backend unificado em Go que substitui a arquitetura de microsservicos legados em PHP 5.6 + CodeIgniter 3 HMVC. O sistema atua como orquestrador central, agendando tarefas e publicando mensagens no RabbitMQ para processamento pelos workers existentes.

## Diagrama de Arquitetura

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                           GESTOR (Go Binary)                                 │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                           main.go                                     │   │
│  │  Carrega config → Conecta banco → Conecta Rabbit → Inicia scheduler │   │
│  └──────────────────────┬───────────────────────────────────────────────┘   │
│                         │                                                   │
│  ┌──────────────────────▼───────────────────────────────────────────────┐   │
│  │                        config.Config                                  │   │
│  │  Hosts, portas, credenciais (env vars)                                │   │
│  └──────────────────────┬───────────────────────────────────────────────┘   │
│                         │                                                   │
│  ┌──────────────────────▼───────────────────────────────────────────────┐   │
│  │                    Conexoes Compartilhadas                             │   │
│  │                                                                        │   │
│  │  ┌──────────────────────┐  ┌──────────────────────────────────────┐  │   │
│  │  │   infra/banco/       │  │   infra/mensageria/                  │  │   │
│  │  │   pool *sql.DB       │  │   *amqp.Connection + *amqp.Channel  │  │   │
│  │  │   (10 conexoes max)  │  │   (monitoramento + reconexao auto)  │  │   │
│  │  └──────────┬───────────┘  └──────────────┬───────────────────────┘  │   │
│  └─────────────┼──────────────────────────────┼──────────────────────────┘   │
│                │                              │                              │
│  ┌─────────────▼──────────────────────────────▼──────────────────────────┐   │
│  │                         cron/agendador.go                              │   │
│  │                                                                        │   │
│  │  robfig/cron v3 (cron.WithSeconds())                                  │   │
│  │                                                                        │   │
│  │   ┌────────────────────────────────────────────────────────────────┐   │   │
│  │   │  7 tarefas registradas                                         │   │   │
│  │   │                                                                 │   │   │
│  │   │  "0 * * * * *"      → cron_1                                   │   │   │
│  │   │  "0 */6 0,3-23 * * *" → run_cluster                            │   │   │
│  │   │  "0 * * * * *"      → check_pop_status                         │   │   │
│  │   │  "0 * * * * *"      → sync_conexoes_radius_arquivo             │   │   │
│  │   │  "0 30 0 * * *"     → repair_radius_acctstoptime               │   │   │
│  │   │  "0 30 0 * * *"     → limpeza_logs                              │   │   │
│  │   │  "0 10 14 * * *"    → listar_clientes_vencidos                  │   │   │
│  │   └────────────────────────────────────────────────────────────────┘   │   │
│  └────────────────────────────────┬───────────────────────────────────────┘   │
│                                   │                                           │
│  ┌────────────────────────────────▼───────────────────────────────────────┐   │
│  │                      cron/tarefas/base.go                               │   │
│  │                                                                        │   │
│  │  Para cada job:                                                        │   │
│  │  1. Ping() no MySQL → se falhar, reconecta                            │   │
│  │  2. BuscarInstanciasAtivas()                                          │   │
│  │  3. Para cada instancia:                                              │   │
│  │     a. Serializar struct → JSON                                       │   │
│  │     b. Codificar em Base64                                            │   │
│  │     c. Publicar na fila RabbitMQ (ate 3 tentativas)                   │   │
│  │  4. Log do resultado                                                  │   │
│  └────────────────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────┬────────────────────────────────────────────┘
                                   │
                                   ▼
                    ┌─────────────────────────────┐
                    │       RabbitMQ Server        │
                    │       172.16.12.10:31837      │
                    │                              │
                    │   ┌─────────────────────┐    │
                    │   │  cron_1             │    │
                    │   │  run_cluster        │    │
                    │   │  check_pop_status   │    │
                    │   │  sync_conexoes...   │    │
                    │   │  repair_radius...   │    │
                    │   │  limpeza_logs       │    │
                    │   │  listar_clientes... │    │
                    │   └─────────────────────┘    │
                    └──────────────┬───────────────┘
                                   │
                                   ▼
                    ┌─────────────────────────────┐
                    │    Workers (Node.js/PHP)     │
                    │                              │
                    │  worker.js                   │
                    │  worker_cron_1.js            │
                    │  worker_kubernets_v2.js      │
                    │  worker2.js                  │
                    │                              │
                    │  Consomem filas e chamam     │
                    │  API PHP para processar      │
                    └─────────────────────────────┘
                                   │
                                   ▼
                    ┌─────────────────────────────┐
                    │    PHP Backend (CodeIgniter) │
                    │                              │
                    │  Webservices.php             │
                    │  cron_1()                    │
                    │  check_pop_status()          │
                    │  limpeza_logs()              │
                    │  ...                         │
                    └─────────────────────────────┘

                    ┌─────────────────────────────┐
                    │   GISPADM (MySQL Central)    │
                    │       177.136.249.51:31034    │
                    │                              │
                    │  instancias (tabela)          │
                    │  ├── id                      │
                    │  ├── token                   │
                    │  ├── env_dbname              │
                    │  ├── env_dbuser              │
                    │  ├── env_dbpass              │
                    │  └── env_dbhost              │
                    └─────────────────────────────┘
```

## Fluxo de Execucao de uma Tarefa

```
Agendador
   │
   ├── [HORA CERTA] Dispara tarefa em goroutine separada
   │
   ├── 1. Logger.Inicio("[fila] Iniciando execucao")
   │
   ├── 2. banco.Ping()                         ← Verifica se MySQL esta vivo
   │      └── Se falhou → reconecta sozinho
   │
   ├── 3. banco.BuscarInstanciasAtivas()       ← SELECT no GISPADM
   │      └── Se falhou → log erro e aborta
   │
   ├── 4. Para cada Instancia:
   │      ├── rabbit.PublicarInstancia()
   │      │     ├── Serializa struct → JSON
   │      │     ├── Base64 encode
   │      │     ├── QueueDeclare (non-durable)
   │      │     └── Publish na fila
   │      │
   │      └── Se falhou → 3 tentativas (1s, 2s, 4s)
   │                      Depois passa para proxima instancia
   │
   └── 5. Logger.Sucesso("[fila] Concluido para N instancias")
```

## Decisoes Tecnicas

### 1. Conexao Unica Compartilhada

Diferente do PHP que abria uma conexao de banco por request, o Go mantem:
- **Pool MySQL**: 10 conexoes max, 5 idle, lifetime 5min
- **Conexao RabbitMQ**: unica, com reconexao automatica via NotifyClose

### 2. Publicacao Direta no RabbitMQ

O cron antigo fazia:
```
PHP → curl → Node.js Producer (3000) → RabbitMQ
```

O novo faz:
```
Go → amqp → RabbitMQ
```

Isso elimina o ponto de falha do Node.js Producer e reduz latencia.

### 3. Resciliencia em Camadas

```
┌───────────────────────────────────────────┐
│            Loop Infinito                   │
│  Conexao inicial: retry 2s..4s..8s..60s  │
│  → Nunca desiste ate conectar             │
├───────────────────────────────────────────┤
│        Reconexao Automatica                │
│  MySQL: Ping() falhou → reconecta         │
│  Rabbit: NotifyClose → reconecta loop     │
├───────────────────────────────────────────┤
│        Retry por Instancia                │
│  Publicacao: 3 tentativas (1s, 2s, 4s)   │
│  → Falha 1 instancia nao quebra o lote   │
└───────────────────────────────────────────┘
```

### 4. Logger Colorido

Sem dependencias externas. Usa codigos ANSI nativos:

| Nivel | Cor | Uso |
|---|---|---|
| INFO | Azul | Mensagens informativas |
| SUCESSO | Verde | Operacoes concluidas |
| AVISO | Amarelo | Tentativas de retry |
| ERRO | Vermelho | Falhas operacionais |
| DESTAQUE | Magenta | Eventos importantes |
| INICIO | Magenta | Inicio de execucao |

### 5. Formato dos Dados Trafegados

O payload publicado no RabbitMQ mantem o formato original do PHP:

```json
{
  "gisp_id": 12,
  "gisp_token": "abc123token",
  "hostname": "177.136.249.51",
  "username": "root",
  "password": "secreta",
  "database": "gisp_cliente"
}
```

Este JSON e codificado em Base64 antes de ser publicado na fila.

## Tecnologias

| Tecnologia | Versao | Uso |
|---|---|---|
| Go | 1.22 | Linguagem principal |
| robfig/cron | v3 | Agendador de tarefas |
| streadway/amqp | v1.1 | Cliente RabbitMQ |
| go-sql-driver/mysql | v1.9 | Driver MySQL |
| RabbitMQ | 3.x | Mensageria (existente) |
| Docker | qualquer | Empacotamento |
| Alpine Linux | 3.19 | Imagem base |

## Seguranca

- **Conexoes MySQL**: autenticacao por usuario/senha
- **Conexoes RabbitMQ**: autenticacao por usuario/senha
- **Dados em transito**: sem criptografia (rede interna)
- **Hardcoded temporario**: as credenciais estao no Dockerfile e serao removidas na Fase 2
