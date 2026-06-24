# SDD-014 — Gateway Pagamentos (Webhook Iugu)

**Status:** Aprovado
**Autor:** Dev Backend
**Prioridade:** Alta
**Dependencias:** Infra existente (`banco`, `fuso`, `logger`, `routeros`)

## 1. Objetivo
Portar o webhook Iugu do PHP (CodeIgniter) para servidor HTTP standalone em Go.
Unico endpoint: `POST /pagamentos/iugu/gatilho/{token}`.

## 2. Arquitetura

```
Iugu -> POST /pagamentos/iugu/gatilho/{token}
         |
   cmd/gateway/main.go  (porta 8082)
         |
   internal/gateway/
     server.go       (mux + graceful shutdown)
     auth.go         (valida token na tabela instancias)
     iugu_webhook.go (parse POST, switch eventos)
     iugu_pagamento.go(processa pagamento, caixa, protocolo, desbloqueio)
     cliente_iugu.go (HTTP client API Iugu)
```

### Fluxo
```
POST /pagamentos/iugu/gatilho/{token}
  -> auth: SELECT FROM instancias WHERE token = ?
  -> banco.ConectarInstancia()
  -> INSERT gisp_iugu_gatilhos
  -> event = invoice.status_changed:
      paid / partially_paid / externally_paid / canceled
  -> Se paid:
      1. SELECT sgp_gateway_pagamentos -> iugu_token
      2. GET https://api.iugu.com/v1/invoices/{id}
      3. UPDATE sgp_clientes_faturas
      4. UPDATE gisp_caixas + INSERT gisp_fluxos_caixas
      5. INSERT protocolo
      6. Se bloqueado: UPDATE status + DEL radreply + RouterOS
      7. UPDATE gisp_iugu_gatilhos SET gisp_exec='1'
  -> 200 OK
```

## 3. Servidor HTTP
- Porta: 8082 (env `GATEWAY_PORT`)
- Graceful shutdown (SIGINT/SIGTERM)
- Pool global GISPADM
- Rota: `POST /pagamentos/iugu/gatilho/{token}`
- Timeout: 60s

## 4. Arquivos novos

| Arquivo | Linhas | Descricao |
|---------|--------|-----------|
| `cmd/gateway/main.go` | ~40 | Entry point, graceful shutdown |
| `internal/gateway/server.go` | ~80 | HTTP server, mux, routes |
| `internal/gateway/auth.go` | ~30 | Validacao de token na tabela `instancias` |
| `internal/gateway/iugu_webhook.go` | ~160 | Parse POST, switch eventos, helpers |
| `internal/gateway/iugu_pagamento.go` | ~400 | Processamento, caixa, protocolo, desbloqueio |
| `internal/gateway/cliente_iugu.go` | ~80 | HTTP client Iugu API |

## 5. Arquivos modificados

| Arquivo | Alteracao |
|---------|-----------|
| `internal/config/config.go` | + `GatewayPort` |
| `internal/infra/banco/gispadm.go` | + `BuscarInstanciaPorToken()` |
| `Dockerfile` | + compilar gateway + entrypoint |

## 6. Tabelas

| Tabela | Op | Banco |
|--------|----|-------|
| `instancias` | SELECT (auth) | gisp_adm |
| `sgp_gateway_pagamentos` | SELECT | instancia |
| `sgp_clientes_faturas` | SELECT + UPDATE | instancia |
| `gisp_iugu_gatilhos` | INSERT + UPDATE | instancia |
| `gisp_iugu_faturas_json` | INSERT + UPDATE | instancia |
| `gisp_caixas` | SELECT + UPDATE | instancia |
| `gisp_fluxos_caixas` | INSERT | instancia |
| `sgp_clientes_contratos_protocolos` | INSERT | instancia |
| `sgp_clientes_contratos` | SELECT + UPDATE | instancia |
| `radreply` | DELETE | instancia |
| `sgp_pop` | SELECT | instancia |

## 7. Helpers PHP -> Go

| PHP | Go |
|-----|-----|
| `date('Y-m-d H:i:s')` | `fuso.Agora().Format("2006-01-02 15:04:05")` |
| `date('Y-m-d')` | `fuso.Agora().Format("2006-01-02")` |
| `date('H:i:s')` | `fuso.Agora().Format("15:04:05")` |
| `gerar_token()` | `fmt.Sprintf("tok_%d", rand.Int63())[:32]` |
| `gerar_protocolo(a,b)` | `a + (counter % (b-a+1))` |
| `var_preg_repleace(s)` | `strings.NewReplacer(".","",",","","R$",""," ","")` |

## 8. Build

```bash
go vet ./cmd/gateway/...
go build ./cmd/gateway
```
