
# Plano de Migracao

## Estado Original

### Arquitetura Legada (PHP 5.6 + CodeIgniter 3)

```
┌─────────────┐    ┌──────────────────┐    ┌─────────────┐
│ ws-cron     │───→│ ws-rabbimq       │───→│ RabbitMQ    │
│ (PHP/Apache)│    │ (PHP Producer)   │    │             │
│             │    │                  │    │  ┌────────┐ │
│ start.sh    │    │ server.js        │    │  │ fila_1 │ │
│ crontab     │    │ (Node Producer)  │    │  │ fila_2 │ │
│             │    │                  │    │  │ ...    │ │
│ Webservices │    │ Webservices.php  │    │  └────────┘ │
│ .php (cron) │    │ (6147 linhas)    │    └──────┬──────┘
└─────────────┘    └──────────────────┘           │
                                          ┌───────▼──────┐
                                          │   Workers    │
                                          │ (Node.js)    │
                                          │              │
                                          │ worker.js    │
                                          │ worker2.js   │
                                          └───────┬──────┘
                                                  │
                                          ┌───────▼──────┐
                                          │ PHP Backend  │
                                          │ (CodeIgniter)│
                                          │              │
                                          │ cron_1()     │
                                          │ check_pop()  │
                                          │ limpeza()    │
                                          └──────────────┘
```

### Problemas da Arquitetura Antiga

1. **5 microsservicos** para manter (cron, producer, workers, gateway, SO)
2. **PHP 5.6** — versao antiga e sem suporte
3. **CodeIgniter 3** — framework legado
4. **Apache** — consumo alto de memoria
5. **Node.js Producer** — ponto de falha desnecessario (só faz forwarding)
6. **Dados hardcoded** — credenciais espalhadas pelos arquivos
7. **Sem resiliencia** — se um servico cai, o processo morre

## Nova Arquitetura

```
┌─────────────────────────────────────────────┐
│              Gestor (Go)                     │
│                                              │
│  ┌──────────────────────────────────────┐   │
│  │  Cron Scheduler (robfig/cron)        │   │
│  │  7 tarefas → publicam no RabbitMQ   │   │
│  └──────────────────┬───────────────────┘   │
│                     │                        │
│  ┌──────────────────▼───────────────────┐   │
│  │  RabbitMQ Publisher (amqp direto)    │   │
│  └──────────────────┬───────────────────┘   │
└─────────────────────┼───────────────────────┘
                      │
              ┌───────▼───────┐
              │   RabbitMQ    │
              │   (7 filas)   │
              └───────┬───────┘
                      │
              ┌───────▼───────┐
              │   Workers     │
              │  (Node.js)    │ ← Fase 2 sera migrado
              └───────┬───────┘
                      │
              ┌───────▼───────┐
              │ PHP Backend   │
              │ (CodeIgniter) │ ← Fase 2 sera substituido
              └───────────────┘
```

## Roadmap em Fases

### Fase 1: Cron Go (ATUAL) ✅

**O que foi feito:**
- Substituiu `gestorisp-ws-cron` (PHP + Apache) por scheduler Go
- Substituiu `gestorisp-ws-rabbimq` (Node.js Producer) — Go publica direto no RabbitMQ
- Implementou resiliencia com retry infinito e reconexao automatica
- Logger colorido para facilitar monitoramento
- Código documentado em portugues

**O que NÃO mudou:**
- Workers continuam em Node.js
- Backend PHP continua processando as requisicoes
- RabbitMQ permanece como infraestrutura
- Dados hardcoded temporariamente no Dockerfile

### Fase 2: Workers Go (PROXIMO) 🔄

**Objetivo:** Migrar os workers Node.js e o backend PHP para Go.

**O que sera feito:**
- Criar consumidores RabbitMQ em `internal/worker/`
- Cada fila tera seu proprio worker (goroutine)
- Migrar logica de negocios do PHP para Go:
  - `cron_1.go` — sync_1(), reparar radius
  - `verificar_pop.go` — check_pop_status via RouterOS API
  - `limpeza_logs.go` — truncate de tabelas
  - `listar_vencidos.go` — bloqueio de clientes
- Implementar conexao com MikroTik (RouterOS API)
- Implementar rotinas de rede (OLT Huawei, SNMP)
- Remover hardcoded do Dockerfile (migrar para Vault ou envs externas)
- Testes unitarios e de integracao

**Estrutura esperada:**
```
internal/
├── worker/                    # Consumidores RabbitMQ
│   ├── base.go                # Consumidor base com reconexao
│   ├── cron_um.go             # Worker da fila cron_1
│   ├── verificar_pop.go       # Worker da fila check_pop_status
│   ├── limpeza_logs.go        # Worker da fila limpeza_logs
│   ├── listar_vencidos.go     # Worker da fila listar_clientes...
│   └── rede/                  # Integracoes de rede
│       ├── mikrotik.go        # RouterOS API
│       ├── radius.go          # FreeRADIUS
│       └── olt_huawei.go     # OLT Huawei SNMP
├── dominio/                   # Entidades adicionais
│   ├── assinante.go
│   ├── contrato.go
│   ├── fatura.go
│   ├── pop.go
│   └── radius.go
```

### Fase 3: API HTTP + Gateway (FUTURO) 📅

**Objetivo:** Substituir `gestorisp-ws-gateway-pagamentos` e `gestorisp-so`.

**O que sera feito:**
- API REST em `internal/api/`
- Gateway de pagamentos (Juno, BoletoFacil, Gerencianet)
- Endpoints para clientes, contratos, faturas
- Autenticacao e autorizacao
- Documentacao OpenAPI/Swagger

## Tabela de Substituicao

| Microsservico | Tecnologia | Fase | Status |
|---|---|---|---|
| `gestorisp-ws-cron` | PHP 5.6 + Apache | Fase 1 | ✅ Substituido |
| `gestorisp-ws-rabbimq` | PHP + Node.js | Fase 1 | ✅ Substituido |
| `RabbitMQ/worker.js` | Node.js | Fase 2 | 🔄 Pendente |
| `RabbitMQ/worker2.js` | Node.js | Fase 2 | 🔄 Pendente |
| `RabbitMQ/worker_cron_1.js` | Node.js | Fase 2 | 🔄 Pendente |
| `RabbitMQ/worker_kubernets*` | Node.js | Fase 2 | 🔄 Pendente |
| `RabbitMQ/worker_single.js` | Node.js | Fase 2 | 🔄 Pendente |
| `gestorisp-ws-gateway-pagamentos` | PHP | Fase 3 | 📅 Pendente |
| `gestorisp-so` | PHP | Fase 3 | 📅 Pendente |

## Melhorias da Migracao

| Aspecto | Antes | Depois |
|---|---|---|
| **Linguagem** | PHP 5.6 + Node.js | Go 1.22 |
| **Servidor** | Apache | Binario unico |
| **Consumo RAM** | ~200MB (Apache + PHP) | ~20MB (Go) |
| **Inicializacao** | ~5s (Apache) | < 1s (Go) |
| **Dependencias** | Debian + PHP + Apache | Alpine + binario |
| **Resiliencia** | Nenhuma | Retry infinito + reconexao |
| **Monitoramento** | Logs simples | Logger colorido estruturado |
| **Build** | Interpretado | Compilado (binary) |
| **Deploy** | Copiar PHP files | Docker image unica |
| **Manutencao** | 5 servicos | 1 binario |

## Tarefas Pendentes (TODO)

### Alta Prioridade
- [ ] Remover variaveis hardcoded do Dockerfile
- [ ] Migrar para provedor de configuracao externo (Vault, Docker Secrets)
- [ ] Implementar testes unitarios para os pacotes
- [ ] Criar pipeline CI/CD

### Media Prioridade
- [ ] Adicionar metricas (Prometheus)
- [ ] Adicionar tracing (OpenTelemetry)
- [ ] Health check endpoint (HTTP :8080/health)

### Baixa Prioridade
- [ ] Configurar log rotativo (log rotation)
- [ ] Adicionar suporte a .env em desenvolvimento
- [ ] Documentacao Swagger para futura API
