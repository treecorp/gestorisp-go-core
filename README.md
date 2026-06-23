
![status](https://img.shields.io/badge/status-em%20desenvolvimento-yellow)
![Go](https://img.shields.io/badge/Go-1.22-blue)
![License](https://img.shields.io/badge/license-MIT-green)

# Gestor ISP - Backend Unificado

Sistema backend unificado em Go para substituir os microsservicos legados em PHP 5.6 + CodeIgniter 3.

## Proposito

Consolidar 5 microsservicos em um unico binario Go, mantendo compatibilidade com o ecossistema existente (RabbitMQ, workers Node.js/PHP) enquanto migra gradualmente toda a logica de negocios para Go.

**Microsservicos sendo substituidos:**

| Microsservico | Funcao | Status |
|---|---|---|
| `gestorisp-ws-cron` | Agendador de tarefas (cron) | ✅ Migrado |
| `gestorisp-ws-rabbimq` | Producer HTTP → RabbitMQ | ✅ Substituido (Go publica direto) |
| `gestorisp-ws-cron` (workers) | Consumidores RabbitMQ | 🔄 Fase 2 |
| `gestorisp-ws-gateway-pagamentos` | Gateway de pagamentos | 📅 Fase 3 |
| `RabbitMQ` (infra) | Mensageria | ✅ Mantido |

## Fluxo de Dados

```
┌──────────────────────────────────────────────────────────────────────┐
│                          Gestor (Go)                                 │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    Cron Scheduler                             │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐           │   │
│  │  │ cron_1  │ │ run_    │ │check_   │ │ ...     │           │   │
│  │  │ */1 min │ │ cluster │ │pop_     │ │ 7 tasks │           │   │
│  │  └────┬────┘ │ */6 min │ │status   │ └─────────┘           │   │
│  │       │      └────┬────┘ │ * * * * │                       │   │
│  │       │           │      └────┬────┘                       │   │
│  │       ▼           ▼           ▼                            │   │
│  │  ┌────────────────────────────────────────────────────┐   │   │
│  │  │          Publisher RabbitMQ (AMQP direto)          │   │   │
│  │  └──────────────────────┬─────────────────────────────┘   │   │
│  └─────────────────────────┼─────────────────────────────────┘   │
│                            │                                      │
│  ┌─────────────────────────┼─────────────────────────────────┐   │
│  │  ┌──────────────────────▼──────────────────────────────┐  │   │
│  │  │         GISPADM (MySQL Central)                     │  │   │
│  │  │  SELECT * FROM instancias WHERE Ativo              │  │   │
│  │  └─────────────────────────────────────────────────────┘  │   │
│  └────────────────────────────────────────────────────────────┘   │
└────────────────────────────┬─────────────────────────────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │    RabbitMQ     │
                    │  (7 filas)      │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │   Workers       │
                    │  (Node.js/PHP)  │
                    │  → processam    │
                    └─────────────────┘
```

## Pre-requisitos

- **Go 1.22+** para desenvolvimento
- **Docker** para empacotamento
- **Acesso de rede** aos servidores:
  - MySQL GISPADM: `177.136.249.51:31034`
  - RabbitMQ: `172.16.12.10:31837`

## Como Executar

### Local (desenvolvimento)

```bash
cd C:\refatoracao_gestor\gestor
go run ./cmd/gestor
```

### Docker

```bash
docker build -t gestor-cron .
docker run --name gestor gestor-cron
```

### Variaveis de Ambiente

| Variavel | Descricao | Padrao |
|---|---|---|
| `DB_GISPADM_HOST` | Host do banco GISPADM | `177.136.249.51` |
| `DB_GISPADM_PORT` | Porta do banco GISPADM | `31034` |
| `DB_GISPADM_USER` | Usuario do banco | `gestorisp` |
| `DB_GISPADM_PASS` | Senha do banco | `WM33223200kl**` |
| `DB_GISPADM_DBNAME` | Nome do banco | `gisp_adm` |
| `RABBITMQ_HOST` | Host do RabbitMQ | `172.16.12.10` |
| `RABBITMQ_PORT` | Porta do RabbitMQ | `31837` |
| `RABBITMQ_USER` | Usuario RabbitMQ | `guest` |
| `RABBITMQ_PASS` | Senha RabbitMQ | `guest` |

> **Atencao:** As variaveis estao hardcoded no `Dockerfile` temporariamente.
> Serao removidas quando a migracao estiver completa.

### Variaveis de Ambiente

```
DB_GISPADM_HOST=177.136.249.51
DB_GISPADM_PORT=31034
DB_GISPADM_USER=gestorisp
DB_GISPADM_PASS="WM33223200kl**"
DB_GISPADM_DBNAME=gisp_adm
RABBITMQ_HOST=172.16.12.10
RABBITMQ_PORT=31837
RABBITMQ_USER=guest
RABBITMQ_PASS=guest
```

## Parar o Sistema

O Gestor responde a sinais `SIGINT` (Ctrl+C) e `SIGTERM`:
1. Finaliza a tarefa cron em execucao
2. Fecha conexao com RabbitMQ
3. Fecha conexao com MySQL
4. Encerra o processo

## Estrutura do Projeto

```
gestor/
├── cmd/
│   └── gestor/
│       └── main.go           # Ponto de entrada
├── internal/
│   ├── config/
│   │   └── config.go         # Configuracoes (env vars)
│   ├── dominio/
│   │   └── instancia.go      # Entidade Instancia
│   ├── infra/
│   │   ├── banco/
│   │   │   ├── mysql.go      # Pool MySQL + reconexao
│   │   │   └── gispadm.go    # Query instancias ativas
│   │   ├── mensageria/
│   │   │   └── rabbit.go     # Publisher RabbitMQ
│   │   └── logger/
│   │       └── logger.go     # Logger colorido
│   ├── cron/
│   │   ├── agendador.go      # Scheduler
│   │   └── tarefas/
│   │       └── base.go       # Logica comum dos jobs
│   ├── worker/               # 🔄 Fase 2
│   │   └── .placeholder
│   └── api/                  # 📅 Fase 3
│       └── .placeholder
├── Dockerfile                # Multi-stage build
├── go.mod
├── go.sum
├── .env.exemplo              # Template de configuracao
└── README.md
```

## Funcionalidades (7 Tarefas Cron)

| Tarefa | Agendamento | Fila RabbitMQ | Descricao |
|---|---|---|---|
| `cron_um` | `*/1 * * * *` | `cron_1` | Tarefa geral de manutencao |
| `executar_cluster` | `*/6 0,3-23 * * *` | `run_cluster` | Atualiza mapa de cluster |
| `verificar_status_pop` | `* * * * *` | `check_pop_status` | Verifica status dos POPs |
| `sincronizar_conexoes` | `* * * * *` | `sync_conexoes_radius_arquivo` | Sincroniza conexoes Radius |
| `reparar_radius` | `30 0 * * *` | `repair_radius_acctstoptime` | Repara registros Radius |
| `limpeza_logs` | `30 0 * * *` | `limpeza_logs` | Limpa logs do sistema |
| `listar_clientes_vencidos` | `10 14 * * *` | `listar_clientes_vencidos` | Bloqueia clientes inadimplentes |

## Roadmap

```
Fase 1 (ATUAL): Cron Go → RabbitMQ
  ├── Substitui ws-cron (PHP)
  ├── Publica direto no RabbitMQ
  └── Resciliencia com retry infinito

Fase 2 (Proxima): Workers Go
  ├── Consomem filas RabbitMQ
  ├── Executam logica de negocio
  └── Substituem worker.js + PHP

Fase 3 (Futuro): API HTTP
  ├── Gateway de pagamentos
  ├── API REST para clientes
  └── Substitui ws-gateway-pagamentos + SO
```

## Licenca

MIT
