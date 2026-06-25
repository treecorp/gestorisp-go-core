# SDD-017 — API REST para Desconexao PPPoE

**Status:** Implementado
**Autor:** Dev Backend
**Prioridade:** Media
**Dependencias:** Infra existente (`mensageria`, `dominio`, `fuso`, `logger`, `config`)

## 1. Objetivo

Criar uma API REST independente que permita disparar desconexoes PPPoE em RouterOS
via publicacao na fila `desconectar_contrato` do RabbitMQ.

A API **nao executa** a desconexao — ela apenas valida os dados e publica na fila.
O worker `desconectar_contrato` ja existente consome a fila e executa a desconexao
com retry infinito.

## 2. Arquitetura

```
Cliente Externo ──POST──▶ API (:8083)
                           └── Validar JSON
                           └── Montar MensagemDesconexaoContrato
                           └── Publicar "desconectar_contrato"
                           └── Return 200

[RabbitMQ] ←─ desconectar_contrato (duravel, persistente, expira 24h)

Worker Desconexao (ja existe)
  └── Decodificar base64 + JSON
  └── Conectar RouterOS
  └── Verificar usuario ativo
  └── Desconectar PPPoE
```

## 3. Endpoint

`POST /api/v2/routeros/desconectarpppoe`

### 3.1 Request Body (JSON)

```json
{
  "instancia_id": 1,
  "contrato_id": 1113,
  "cliente_nome": "Fulano de Tal",
  "pppoe_user": "fulano@isp",
  "pop_ipv4": "177.136.249.55",
  "pop_port": "8728",
  "pop_user": "admin",
  "pop_pass": "senha"
}
```

### 3.2 Response (200 — Sucesso)

```json
{
  "sucesso": true,
  "mensagem": "Publicado na fila desconectar_contrato"
}
```

### 3.3 Response (400 — Erro de validacao)

```json
{
  "sucesso": false,
  "erro": "pppoe_user é obrigatorio"
}
```

### 3.4 Response (500 — Erro interno)

```json
{
  "sucesso": false,
  "erro": "Erro ao publicar na fila: ..."
}
```

## 4. Campos Obrigatorios

| Campo | Tipo | Obrigatorio | Descricao |
|-------|------|-------------|-----------|
| `pppoe_user` | string | Sim | Login PPPoE do cliente |
| `pop_ipv4` | string | Sim | IP do RouterOS |
| `pop_port` | string | Sim | Porta API RouterOS (padrao 8728) |
| `pop_user` | string | Sim | Usuario RouterOS |
| `pop_pass` | string | Sim | Senha RouterOS |
| `instancia_id` | int | Nao | ID da instancia GISP (log apenas) |
| `contrato_id` | int | Nao | ID do contrato (log apenas) |
| `cliente_nome` | string | Nao | Nome do cliente (log apenas) |

## 5. Fluxo

1. Cliente faz POST com JSON
2. Servidor valida campos obrigatorios
3. Monta `MensagemDesconexaoContrato` com `CriadoEm = fuso.Agora().Format(time.RFC3339)`
4. Publica na fila `desconectar_contrato` via `rabbit.PublicarMensagem`
5. Retorna `{"sucesso": true}` (HTTP 200)
6. Em caso de erro: HTTP 400 (validacao) ou 500 (fila)

## 6. Binario Separado

A API roda em um binario independente (`cmd/api`), separado do gateway de pagamentos:

- Gateway (Iugu): porta `8082`
- API (Desconexao): porta `8083`

Vantagens:
- Escalam independentemente
- Um nao afeta o outro
- Portas diferentes para load balancing diferente

## 7. Configuracao

- Porta: env `API_PORT`, default `8083`
- Sem autenticacao (uso interno)
- Sem conexao com banco — apenas RabbitMQ

## 8. Arquivos Envolvidos

| Arquivo | Acao |
|---------|------|
| `cmd/api/main.go` | Criar — entrypoint do binario |
| `internal/api/routeros_handler.go` | Criar — handler do endpoint |
| `internal/config/config.go` | Alterar — adicionar campo `APIPort` |
| `Dockerfile` | Alterar — adicionar build + copy + entrypoint case |
| `.opencode/memory/index.md` | Alterar — update stats |
| `.opencode/specs/sdd-017-api-rest-desconexao-pppoe.md` | Este arquivo |
