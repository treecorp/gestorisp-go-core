# Estrutura de Diretórios (Pós-Refatoração)

## Árvore Completa

```
gestor/
├── cmd/                              ← Entrypoints (binários)
│   ├── gestor/main.go                Cron scheduler (broker + tarefas agendadas)
│   ├── worker/main.go                RabbitMQ consumer independente
│   ├── gateway/main.go               HTTP gateway Iugu (porta 8082)
│   ├── api/main.go                   HTTP REST API (porta 8083)
│   └── testedesconexao/main.go       Ferramenta CLI para teste RouterOS
│
├── internal/                         ← Código privado do projeto
│   │
│   ├── entity/                       ← Entidades de domínio (Model)
│   │   ├── contrato.go               Contrato de cliente (ID, status, dados PPPoE)
│   │   ├── desconexao.go             MensagemDesconexaoContrato (payload fila)
│   │   ├── fatura.go                 Fatura de cliente (vencimento, valor, status)
│   │   ├── instancia.go              Instância GISP (credenciais de banco)
│   │   ├── pagamento.go              MensagemPagamentoIugu (payload webhook)
│   │   └── pop.go                    POP — Ponto de Presença (dados RouterOS)
│   │
│   ├── helpers/                      ← Funções puras e utilitárias (Helper)
│   │   ├── data.go                   ExtrairData, ExtrairHora de timestamp
│   │   ├── gerar_token.go            GerarToken usando crypto/rand
│   │   ├── moeda.go                  FormatarMoeda, LimparNumero (string → centavos)
│   │   ├── protocolo.go              GerarProtocolo (sequencial por instância)
│   │   └── string.go                 Truncate, manipulação de strings
│   │
│   ├── lib/                          ← Bibliotecas de integração externa
│   │   └── iugu/
│   │       ├── cliente.go            Cliente HTTP para API de Fatura da Iugu
│   │       └── doc.go                Documentação do pacote
│   │
│   ├── repositorio/                  ← Acesso a dados (Model — queries SQL)
│   │   ├── bloqueio_repo.go          Queries de bloqueio (aplicar/remover)
│   │   ├── cluster_repo.go           Queries de cluster/coordenadas
│   │   ├── contrato_repo.go          CRUD de contratos (cliente, status, PPPoE)
│   │   ├── fatura_repo.go            CRUD de faturas (vencidas, por token)
│   │   ├── gatilho_repo.go           Insert/select de gatilhos Iugu
│   │   ├── pop_repo.go               Queries de POPs (endereço RouterOS)
│   │   └── radacct_repo.go           Queries RADIUS accounting (AcctStopTime)
│   │
│   ├── service/                      ← Regras de negócio (Service)
│   │   ├── pagamento/                Processamento de pagamentos Iugu
│   │   │   ├── baixa.go              Lógica de baixa financeira (lançar caixa)
│   │   │   ├── contrato.go           Desbloqueio de contrato pós-pagamento
│   │   │   ├── origem.go             Mapa de origens de pagamento
│   │   │   ├── processar.go          Orquestrador principal (ProcessarPagamento)
│   │   │   └── service.go            Interface/struct do service
│   │   │
│   │   └── bloqueio/                 Bloqueio por inadimplência
│   │       ├── cliente.go            Lógica: ProcessarFatura, DeveBloquear
│   │       └── service.go            Interface/struct do service
│   │
│   ├── handler/                      ← Controladores (Controller)
│   │   ├── gateway/                  HTTP gateway Iugu — porta 8082
│   │   │   ├── auth.go               Autenticação por token de instância
│   │   │   ├── server.go             Servidor HTTP (roteamento, middleware)
│   │   │   └── webhook.go            Handler POST /pagamentos/iugu/gatilho/{token}
│   │   │
│   │   ├── api/                      REST API — porta 8083
│   │   │   ├── errors.go             Definição de erros padronizados
│   │   │   ├── openapi.yaml          Especificação OpenAPI da API
│   │   │   ├── routeros.go           Handler de desconexão RouterOS
│   │   │   ├── server.go             Servidor HTTP (roteamento, middleware)
│   │   │   ├── swagger.go            Servir Swagger UI estático
│   │   │   └── webhook_iugu.go       Webhook alternativo Iugu pela API
│   │   │
│   │   ├── worker/                   Consumidores RabbitMQ
│   │   │   ├── check_pop_status.go           Job: verificar status dos POPs
│   │   │   ├── consumidor.go                 Configuração de filas/consumidores
│   │   │   ├── consumidor_mensagem.go        Interface ConsumidorMensagem
│   │   │   ├── cron_1.go                     Job: cron_1 genérico
│   │   │   ├── desconectar_contrato.go       Job: desconexão PPPoE via RouterOS
│   │   │   ├── limpeza_logs.go               Job: limpeza de logs antigos
│   │   │   ├── listar_clientes_vencidos.go   Job: bloqueio de inadimplentes
│   │   │   ├── processar_pagamento_iugu.go   Job: processar pagamento Iugu
│   │   │   ├── repair_radius_acctstoptime.go Job: reparar radacct órfão
│   │   │   ├── run_cluster.go               Job: execução de cluster
│   │   │   ├── sync_conexoes_radius_arquivo.go Job: sincronia RADIUS
│   │   │   ├── wiring.go                     Injeção de dependências dos jobs
│   │   │   └── worker.go                     Loop principal do worker
│   │   │
│   │   └── cron/                     Agendador de tarefas
│   │       ├── agendador.go          Scheduler robfig/cron + registro de jobs
│   │       └── tarefas/
│   │           ├── base.go           Job base: ping → instâncias → publicar fila
│   │           └── doc.go            Documentação do pacote
│   │
│   └── infra/                        ← Infraestrutura base
│       ├── banco/                    Pool MySQL + queries de sistema
│       │   ├── gispadm.go            Queries do banco GISPADM (instâncias)
│       │   ├── gispinstancia.go      Queries específicas de instância
│       │   └── mysql.go              Pool de conexões (10 max), Ping, ConectarComRetry
│       │
│       ├── cripto/                   Criptografia
│       │   ├── ci3.go               Algoritmo compatível com CodeIgniter 3
│       │   └── hkdf.go              HKDF key derivation
│       │
│       ├── fuso/                     Fuso horário
│       │   └── fuso.go               Timezone America/Sao_Paulo
│       │
│       ├── logger/                   Logger colorido ANSI
│       │   └── logger.go             Info, Sucesso, Aviso, Erro, Destaque, Inicio
│       │
│       ├── mensageria/               Mensageria RabbitMQ
│       │   └── rabbit.go             Conexão, publisher, reconexão automática
│       │
│       └── routeros/                 Cliente RouterOS API
│           └── client.go             Conexão SSH + comando de desconexão PPPoE
│
├── docs/                             ← Documentação
│   ├── arquitetura.md                Arquitetura em camadas + fluxos
│   ├── estrutura.md                  Este arquivo
│   └── diagrama.md                   Diagramas de sequência Mermaid
│
├── .github/workflows/                CI/CD (GitHub Actions)
├── .opencode/                        Configuração opencode + specs + memória
│   ├── memory/                       Banco de Memória do Projeto
│   ├── plans/                        Planos de execução
│   └── specs/                        SDDs (especificações)
│
├── Dockerfile                        Build multi-stage unificado
├── go.mod                            Módulo Go
├── go.sum                            Checksum de dependências
├── README.md                         Documentação principal
└── .env.exemplo                      Template de variáveis de ambiente
```

---

## Convenções do Projeto

### Nomenclatura

| Elemento | Convenção | Exemplo |
|----------|-----------|---------|
| Pacotes | português, minúsculo, sem underlines | `repositorio`, `pagamento` |
| Arquivos | português, minúsculo, snake_case | `fatura_repo.go`, `gerar_token.go` |
| Funções exportadas | PascalCase | `ProcessarPagamento`, `DeveBloquear` |
| Funções privadas | camelCase | `processarIuguDireto`, `publicarComRetry` |
| Variáveis | camelCase | `faturaID`, `contratoAtivo` |
| Constantes | PascalCase | `StatusAtivo`, `TipoBloqueio` |
| Interfaces | sufixo `er` ou domínio | `RepositorioFatura`, `ConsumidorMensagem` |

### Tratamento de Erros

- Erros são propagados com `fmt.Errorf("contexto: %w", err)` — contexto em português
- Funções de infra retornam `error` para a camada superior decidir
- Logging usa exclusivamente `logger.Info/Sucesso/Aviso/Erro` — nunca `fmt.Print`/`log.Println`
- Erros em lote não interrompem o resto do lote
- Conexões usam retry infinito com backoff exponencial (2s → 4s → ... → 60s)
- O sistema **nunca** pode cair — sem `os.Exit` ou `panic` recuperável

### Organização de Pacotes

- `repositorio/`: cada arquivo mapeia uma entidade (`fatura_repo.go`, `contrato_repo.go`)
- `service/`: cada subpacote é um domínio de negócio (`pagamento/`, `bloqueio/`)
- `handler/`: cada subpacote é um ponto de entrada (`gateway/`, `api/`, `worker/`, `cron/`)
- `entity/`: um arquivo por entidade, todas no mesmo pacote `entity`
- `helpers/`: um arquivo por categoria de função (`moeda.go`, `string.go`, `data.go`)
