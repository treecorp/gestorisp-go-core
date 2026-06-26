# SDD-000 — Documentação de Arquitetura

**Status:** Pendente
**Autor:** Knowledge Engineer
**Prioridade:** Média
**Dependencias:** SDD-018, SDD-019, SDD-020, SDD-021, SDD-022, SDD-023, SDD-024, SDD-025, SDD-026

## 1. Objetivo

Criar a documentação de arquitetura completa do sistema após a refatoração, descrevendo a estrutura em camadas, os fluxos de dados e a organização dos pacotes. Esta documentação serve como referência para novos desenvolvedores e para agentes de IA.

## 2. Escopo

### 2.1 docs/arquitetura.md

Diagrama Mermaid da arquitetura em camadas + descrição de cada camada:

```mermaid
graph TD
    subgraph "Handler (Transporte)"
        G[handler/gateway]
        A[handler/api]
        W[handler/worker]
        C[handler/cron]
    end

    subgraph "Service (Negócio)"
        SP[service/pagamento]
        SB[service/bloqueio]
    end

    subgraph "Repositorio (Dados)"
        RF[fatura_repo]
        RC[contrato_repo]
        RG[gatilho_repo]
        RB[bloqueio_repo]
        RP[pop_repo]
        RR[radacct_repo]
        RCL[cluster_repo]
    end

    subgraph "Lib (Integração)"
        LI[lib/iugu]
    end

    subgraph "Entity (Domínio)"
        EI[entity/instancia]
        EC[entity/contrato]
        EF[entity/fatura]
        EP[entity/pagamento]
        ED[entity/desconexao]
        EPO[entity/pop]
    end

    subgraph "Helpers (Utilitários)"
        HM[helpers/moeda]
        HS[helpers/string]
        HP[helpers/protocolo]
        HD[helpers/data]
        HG[helpers/gerar_token]
    end

    G --> SP
    A --> SP
    W --> SP
    W --> SB
    C --> SP
    C --> SB

    SP --> RF
    SP --> RC
    SP --> RG
    SB --> RB
    SB --> RF
    SB --> RC

    G --> LI
    SP --> LI

    RF --> EF
    RC --> EC
    RB --> EF
    RP --> EPO

    SP --> HM
    SP --> HS
    SP --> HP
    SB --> HD
    LI --> HG
```

### 2.2 docs/estrutura.md

Árvore de diretórios completa após refatoração:

```
gestor/
├── cmd/
│   ├── gestor/main.go          <- Entrypoint principal (cron + worker)
│   ├── gateway/main.go         <- Gateway Iugu (porta 8082)
│   ├── api/main.go             <- API REST (porta 8083)
│   └── worker/main.go          <- Worker独立 (processamento filas)
├── internal/
│   ├── entity/                 <- Tipos de domínio enriquecidos
│   │   ├── instancia.go
│   │   ├── contrato.go
│   │   ├── fatura.go
│   │   ├── pagamento.go
│   │   ├── desconexao.go
│   │   └── pop.go
│   ├── helpers/                <- Funções puras e utilitárias
│   │   ├── moeda.go
│   │   ├── string.go
│   │   ├── protocolo.go
│   │   ├── data.go
│   │   └── gerar_token.go
│   ├── lib/                    <- Bibliotecas de integração externa
│   │   └── iugu/cliente.go
│   ├── repositorio/            <- Acesso a dados (Repository pattern)
│   │   ├── fatura_repo.go
│   │   ├── contrato_repo.go
│   │   ├── gatilho_repo.go
│   │   ├── bloqueio_repo.go
│   │   ├── pop_repo.go
│   │   ├── radacct_repo.go
│   │   └── cluster_repo.go
│   ├── service/                <- Regras de negócio
│   │   ├── pagamento/
│   │   │   ├── processar.go
│   │   │   ├── baixa.go
│   │   │   ├── contrato.go
│   │   │   └── origem.go
│   │   └── bloqueio/
│   │       └── cliente.go
│   ├── handler/                <- Transporte HTTP / handlers
│   │   ├── gateway/
│   │   ├── api/
│   │   ├── worker/
│   │   └── cron/
│   ├── config/config.go        <- Configurações (env vars)
│   └── infra/                  <- Infraestrutura (banco, fila, log)
│       ├── banco/
│       ├── mensageria/
│       └── logger/
├── docs/                       <- Documentação
│   ├── arquitetura.md
│   ├── estrutura.md
│   └── diagrama.md
├── .opencode/                  <- Config + memória do projeto
│   ├── specs/                  <- SDDs
│   ├── memory/                 <- Banco de Memória do Projeto
│   └── plans/                  <- Planos de execução
└── Dockerfile
```

### 2.3 docs/diagrama.md

Diagramas Mermaid dos principais fluxos:

**Fluxo de Webhook Iugu:**

```mermaid
sequenceDiagram
    participant Iugu as API Iugu
    participant GW as handler/gateway
    participant Auth as auth.go
    participant Svc as service/pagamento
    participant Repo as repositorio
    participant DB as MySQL

    Iugu->>GW: POST /pagamentos/iugu/gatilho/{token}
    GW->>Auth: Autenticar(token)
    Auth->>DB: SELECT instancia
    DB-->>Auth: Instancia
    Auth-->>GW: Instancia válida
    GW->>Svc: ProcessarPagamento(repo, iugu, instancia, data)
    Svc->>Repo: BuscarFaturaPorToken(db, token)
    Repo->>DB: SELECT fatura
    DB-->>Repo: Fatura
    Svc->>Repo: BuscarContratoPorID(db, id)
    Repo->>DB: SELECT contrato
    DB-->>Repo: Contrato
    Svc->>Svc: processarIuguDireto()
    Svc->>Repo: AtualizarStatusFatura(tx, id, status, protocolo)
    Svc->>Repo: DesbloquearContrato(tx, id)
    Svc-->>GW: ResultadoBaixa
    GW-->>Iugu: 200 JSON
```

**Fluxo de Cron (Bloqueio):**

```mermaid
sequenceDiagram
    participant Cron as handler/cron
    participant Svc as service/bloqueio
    participant Repo as repositorio
    participant DB as MySQL

    Cron->>Cron: Ticker (ex: 5min)
    Cron->>Repo: BuscarFaturasVencidas(db)
    Repo->>DB: SELECT faturas + contratos
    DB-->>Repo: []Fatura
    loop Cada fatura
        Cron->>Svc: ProcessarFatura(repo, tag, db, fatura, dias)
        Svc->>Svc: DeveBloquear(fatura, contrato, dias)
        alt Deve bloquear
            Svc->>Repo: AtualizarStatusContrato(db, id, "bloqueado")
            Svc-->>Cron: ClienteBloqueado
        else Não deve bloquear
            Svc-->>Cron: nil
        end
    end
```

**Fluxo de Worker (Desconexão):**

```mermaid
sequenceDiagram
    participant RB as RabbitMQ
    participant W as handler/worker
    participant Repo as repositorio
    participant DB as MySQL
    participant ROS as RouterOS

    RB->>W: Mensagem "desconectar_contrato"
    W->>W: Parse mensagem
    W->>Repo: BuscarContratoPorID(db, id)
    Repo->>DB: SELECT contrato
    DB-->>Repo: Contrato
    W->>ROS: /tool/ssh {user}@{pop} disconnect {pppoe_user}
    ROS-->>W: OK
    W->>DB: UPDATE radacct SET AcctStopTime = NOW()
    W-->>RB: Ack
```

## 3. Camadas da Arquitetura

### 3.1 Entity (internal/entity)

Tipos de domínio enriquecidos com métodos de negócio. Não contêm dependências externas. São os blocos fundamentais do sistema.

### 3.2 Helpers (internal/helpers)

Funções puras e utilitárias sem estado (exceto `GerarProtocolo` que usa contador global). Não dependem de nenhum outro pacote interno.

### 3.3 Lib (internal/lib)

Clientes para APIs externas (Iugu). Encapsulam chamadas HTTP, serialização e lógica de integração.

### 3.4 Repositório (internal/repositorio)

Acesso a dados seguindo o padrão Repository. Cada arquivo agrupa queries relacionadas a uma entidade. Funções recebem `*sql.DB` ou `*sql.Tx` explicitamente.

### 3.5 Service (internal/service)

Regras de negócio puras. Services dependem de interfaces (não de implementações concretas de repositório). Não têm acesso direto a `*sql.DB`.

### 3.6 Handler (internal/handler)

Camada de transporte. Handlers fazem parse de requests, delegam para services e formatam respostas. Não contêm lógica de negócio ou SQL.

## 4. Arquivos a criar

| Arquivo | Ação |
|---------|------|
| `docs/arquitetura.md` | Criar — diagrama + descrição das camadas |
| `docs/estrutura.md` | Criar — árvore + explicação dos pacotes |
| `docs/diagrama.md` | Criar — fluxos (webhook, cron, worker) |

## 5. Critérios de Aceite

- [ ] `docs/arquitetura.md` contém diagrama Mermaid funcional
- [ ] `docs/estrutura.md` reflete a estrutura real pós-refatoração
- [ ] `docs/diagrama.md` documenta os 3 fluxos principais
- [ ] Toda documentação em português
- [ ] Documentos aprovados pelo time técnico
