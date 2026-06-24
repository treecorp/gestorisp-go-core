# HOTFIX-001 — Parse PHP-style do webhook Iugu + logs enriquecidos

**Status:** Aprovado
**Autor:** Dev Backend
**Prioridade:** Critica
**Tipo:** Correcao

## 1. Problema

O webhook Iugu envia os dados no formato **PHP-style**:

```
POST /pagamentos/iugu/gatilho/{token}
Content-Type: application/x-www-form-urlencoded

event=invoice.status_changed&data[id]=899E76B4&data[status]=paid&data[payer_name]=IGOR&data[external_reference]=abc123
```

No PHP, `$_POST['data']` vira automaticamente um array associativo. No Go, `r.PostFormValue("data")` procura a chave literal `"data"` — **que nao existe** — e retorna vazio, causando **HTTP 400** em todos os webhooks.

## 2. Diagnostico

Logs reais mostram:
```
[gateway]  Request recebido para token 237468723rgjhfgjsdgf763t4r34
[gateway]  Instancia 1 autenticada: gisp_isp_local_1
(sem mais logs - 400 silencioso)
```

O handler atual tenta:
1. `r.PostFormValue("data")` → `""` (vazio)
2. `json.Unmarshal("", &data)` → erro
3. Retorna 400 sem log

## 3. Correcao

### 3.1 Parse PHP-style em `iugu_webhook.go`

Substituir o parse JSON por parse manual de chaves `data[chave]`:

```go
r.ParseForm()
event := r.PostFormValue("event")

data := make(map[string]string)
for key, values := range r.Form {
    if strings.HasPrefix(key, "data[") && strings.HasSuffix(key, "]") {
        campo := key[5 : len(key)-1]
        if len(values) > 0 {
            data[campo] = values[0]
        }
    }
}
```

### 3.2 Tipo `data` muda de `map[string]interface{}` para `map[string]string`

Remove todos os `valorString(data, "chave")` e type assertions:
- `iugu_webhook.go` — `handleStatusChanged()` acessa `data["id"]` direto (ja e string)
- `iugu_pagamento.go` — `processarPagamento()` e demais recebem `map[string]string`

### 3.3 Logs enriquecidos em todos os pontos

| Arquivo | Local | Log |
|---------|-------|-----|
| `iugu_webhook.go` | HandleWebhook | `Webhook: instancia=%d event=%s fatura_iugu=%s status=%s ref=%s pagador=%s` |
| `iugu_pagamento.go` | processarPagamentoIuguDireto | `Fatura encontrada: id=%d contrato=%d valor=%s status=%s` |
| `iugu_pagamento.go` | processarPagamentoIuguDireto | `Fatura %d ja estava paga (contrato=%d valor=%s)` |
| `iugu_pagamento.go` | executarBaixa | `Baixando fatura %d: contrato=%d cliente=%s valor=%s origem=%s` |
| `iugu_pagamento.go` | executarBaixa | `Fatura %d baixada: contrato=%d cliente=%s protocolo=%s` |
| `iugu_pagamento.go` | desbloquearContrato | `Contrato %d desbloqueado: pppoe=%s pop=%d` |
| `iugu_pagamento.go` | buscarContrato | `Contrato %d: cliente=%s pppoe=%s pop=%d` |

### 3.4 Adicionar `ClienteNome` no `contratoRow`

```go
type contratoRow struct {
    ID          int
    Token       string
    Status      string
    ClienteID   int
    ClienteNome string
    ClienteToken string
    PopID       int
    PPPoEUser   string
}
```

Modificar `buscarContrato` para JOIN com `sgp_clientes_new`:

```sql
SELECT c.id, c.token, c.status, c.cliente_id, 
       COALESCE(cli.pf_nome, cli.pj_razao_social, 'N/D') AS cliente_nome,
       c.cliente_token, c.pop_id, c.pppoe_user
FROM sgp_clientes_contratos c
LEFT JOIN sgp_clientes_new cli ON cli.id = c.cliente_id
WHERE c.id = ?
```

## 4. Arquivos Modificados

| Arquivo | Alteracao |
|---------|-----------|
| `internal/gateway/iugu_webhook.go` | Parse PHP-style + tipo `map[string]string` + logs |
| `internal/gateway/iugu_pagamento.go` | `contratoRow` + `buscarContrato` + logs em todos os pontos |
| `internal/gateway/cliente_iugu.go` | Nenhuma (nao precisa mudar) |
| `internal/gateway/server.go` | Nenhuma |
| `internal/gateway/auth.go` | Nenhuma |

## 5. Nao alterado

- Logica de negocio (fatura, caixa, protocolo, desbloqueio) — **zero mudancas**
- Queries SQL (exceto `buscarContrato` que ganha JOIN)
- API Iugu cliente
- Autenticacao
- Dockerfile / CI

## 6. Validacao

```bash
go vet ./internal/gateway/...
go build ./cmd/gateway
```

Teste manual:
```bash
curl -X POST http://localhost:8082/pagamentos/iugu/gatilho/TOKEN \
  -d "event=invoice.status_changed" \
  -d "data[id]=F123" \
  -d "data[status]=paid" \
  -d "data[external_reference]=abc123" \
  -d "data[payer_name]=Teste"
```

Esperado: log com `Webhook: event=invoice.status_changed status=paid ref=abc123 pagador=Teste` + processamento normal (ou erro contextualizado se fatura nao existir).
