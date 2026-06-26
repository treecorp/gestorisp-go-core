# Diagramas de Sequência

## 1. Webhook Iugu (Processar Pagamento)

Fluxo completo: Iugu envia webhook HTTP → gateway autentica → publica na fila →
worker consome → service processa → repositorio persiste.

```mermaid
sequenceDiagram
    participant Iugu as Iugu
    participant GW as handler/gateway
    participant MQ as RabbitMQ
    participant WK as handler/worker
    participant SV as service/pagamento
    participant DB as MySQL
    participant IUGUAPI as Iugu API

    Iugu->>GW: POST /pagamentos/iugu/gatilho/{token}
    GW->>DB: Autenticar token (SELECT instancia)
    DB-->>GW: Instancia válida
    GW->>MQ: Publicar processar_pagamento_iugu
    GW-->>Iugu: 200 OK (resposta imediata)

    MQ->>WK: Consumir mensagem da fila
    WK->>SV: ProcessarPagamento(repos, iuguCli, instancia, payload)
    SV->>IUGUAPI: ConsultarFatura(id) — API Iugu
    IUGUAPI-->>SV: FaturaIugu (dados do pagamento)
    SV->>SV: Validar valor, status, origem
    SV->>DB: InserirGatilho (tx)
    SV->>DB: BuscarFatura por token (tx)
    SV->>DB: AtualizarStatusFatura (pago, protocolo)
    SV->>DB: LancarCaixa (tx)
    SV->>DB: CriarProtocolo (tx)
    alt Fatura paga com sucesso
        SV->>DB: DesbloquearContrato (tx)
        SV-->>WK: ResultadoBaixa{Sucesso: true}
        WK->>MQ: Publicar desconectar_contrato (se necessário)
    else Erro no processamento
        SV-->>WK: ResultadoBaixa{Sucesso: false, Erro}
        WK->>MQ: Publicar para DLQ (dead letter)
    end
    WK-->>MQ: Ack da mensagem
```

**Observações:**
- O gateway responde `200` imediatamente para a Iugu, antes do processamento
- O processamento real é assíncrono via RabbitMQ
- A transação de banco garinte atomicidade entre baixa, atualização e desbloqueio

---

## 2. Cron — Listar Clientes Vencidos (Bloqueio)

O cron scheduler executa periodicamente (a cada 5 min) para detectar faturas vencidas
e aplicar bloqueio nos contratos inadimplentes.

```mermaid
sequenceDiagram
    participant CR as handler/cron
    participant RP as repositorio
    participant DB as MySQL
    participant SV as service/bloqueio
    participant MQ as RabbitMQ

    Note over CR: Ticker a cada 5 minutos
    CR->>RP: BuscarFaturasVencidas(db)
    RP->>DB: SELECT faturas + contratos
    DB-->>RP: []Fatura (com contrato aninhado)
    RP-->>CR: []Fatura

    loop Para cada fatura vencida
        CR->>SV: ProcessarFatura(repos, instancia, fatura, diasTolerancia)
        SV->>SV: DeveBloquear(fatura, contrato, dias)
        SV->>RP: LerContrato(db, contratoID)
        RP-->>SV: Contrato (status, dados PPPoE)
        SV->>RP: LerDesbloqueioConfianca(db, contratoID)
        RP-->>SV: DesbloqueioAtivo?

        alt Deve bloquear
            SV->>RP: AplicarBloqueio(tx)
            RP->>DB: UPDATE contrato status='bloqueado'
            RP->>DB: INSERT historico_bloqueio
            DB-->>RP: OK
            RP-->>SV: ClienteBloqueado
            SV-->>CR: Resultado{ClienteBloqueado, Contrato}
            CR->>MQ: Publicar desconectar_contrato
        else Não deve bloquear (toledo, desbloqueio confiança ativo)
            SV-->>CR: nil
        end
    end

    Note over CR: Próximo tick em 5 minutos
```

**Regras de bloqueio:**
- Dias de tolerância configurados por instância
- Contratos com desbloqueio de confiança ativo são ignorados
- Bloqueio só ocorre se fatura vencida há mais de N dias

---

## 3. Fluxo de Desconexão RouterOS (via Worker)

Quando um contrato precisa ser desconectado (pagamento pendente ou bloqueio),
uma mensagem é publicada no RabbitMQ e o worker executa a desconexão via SSH.

```mermaid
sequenceDiagram
    participant MQ as RabbitMQ
    participant WK as handler/worker
    participant RP as repositorio
    participant DB as MySQL
    participant ROS as RouterOS (Mikrotik)

    MQ->>WK: Mensagem "desconectar_contrato"
    WK->>WK: Parse payload (contratoID, instancia)
    WK->>RP: BuscarContratoPorID(db, contratoID)
    RP->>DB: SELECT contrato (dados PPPoE, POP)
    DB-->>RP: Contrato
    RP-->>WK: Contrato (pppoe_user, pop_endereco)

    WK->>RP: BuscarPOPPorID(db, popID)
    RP->>DB: SELECT pop (host, usuario, porta SSH)
    DB-->>RP: POP
    RP-->>WK: POP (endereço IP, credenciais)

    WK->>ROS: SSH connect {pop.host}:{pop.porta}
    ROS-->>WK: Connected
    WK->>ROS: /tool/ssh {pppoe_user}@{pop.host} disconnect
    ROS-->>WK: OK (usuário desconectado)

    WK->>DB: UPDATE radacct SET AcctStopTime = NOW()
    WK->>DB: INSERT log_desconexao
    WK-->>MQ: Ack da mensagem

    Note over WK: Se falhar, retry com backoff (3 tentativas)
```

**Resiliência:**
- 3 tentativas de reconexão SSH (1s, 2s, 4s de intervalo)
- Se todas falharem, a mensagem vai para DLQ para inspeção manual
- Log detalhado de cada tentativa (logger.Aviso)

---

## 4. Fluxo de Boot (Inicialização dos Componentes)

Como os entrypoints inicializam o sistema e conectam as dependências.

```mermaid
sequenceDiagram
    participant Main as cmd/main.go
    participant Cfg as config
    participant DB as infra/banco
    participant MQ as infra/mensageria
    participant CR as handler/cron
    participant WK as handler/worker
    participant SV as service
    participant RP as repositorio

    Main->>Cfg: CarregarConfig() — lê env vars
    Cfg-->>Main: Config{}

    Main->>DB: ConectarComRetry(cfg.Banco)
    loop backoff exponencial 2s→4s→...→60s
        DB->>DB: tentar conexão MySQL
    end
    DB-->>Main: *sql.DB (pool de conexões)

    Main->>MQ: ConectarComRetry(cfg.RabbitMQ)
    loop backoff exponencial 2s→4s→...→60s
        MQ->>MQ: tentar conexão RabbitMQ
    end
    MQ-->>Main: *amqp.Connection + *amqp.Channel

    Main->>RP: Criar repositórios (recebem *sql.DB)
    Main->>SV: Criar services (recebem interfaces de repositório)
    Main->>WK: Criar worker (recebe services + MQ)
    Main->>CR: Criar cron (recebe services + MQ + DB)
    Main->>CR: Iniciar() — registra jobs e começa a execução

    Note over Main: Graceful shutdown: aguarda SIGINT/SIGTERM
    Main->>CR: Parar() — aguarda jobs em execução
    Main->>MQ: Fechar()
    Main->>DB: Fechar()
    Note over Main: Processo encerra com código 0
```

**Entrypoints específicos:**

| Binary | Inicializa | Porta | Dependências |
|--------|------------|-------|-------------|
| `cmd/gestor` | cron scheduler | — | DB + MQ + repos + services |
| `cmd/worker` | consumer RabbitMQ | — | DB + MQ + repos + services |
| `cmd/gateway` | HTTP server Iugu | 8082 | DB + MQ + services |
| `cmd/api` | HTTP REST API | 8083 | DB + MQ + services + infra/routeros |
