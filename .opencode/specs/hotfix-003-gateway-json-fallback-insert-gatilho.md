# HOTFIX-003 — Inserção gisp_iugu_gatilhos + Fallback JSON no Gateway

**Status:** Aprovado
**Autor:** Dev Backend
**Prioridade:** Crítica
**Tipo:** Correcao

## 1. Problema 1 — INSERT gisp_iugu_gatilhos ausente (CRÍTICO)

Quando o gateway foi convertido para assíncrono (SDD-015), a função 
`handleStatusChanged()` que fazia o INSERT em `gisp_iugu_gatilhos` foi removida.

O worker `processar_pagamento_iugu` só fazia SELECT/UPDATE — a linha 
nunca era criada. Resultado: **pagamento em dobro** se Iugu reenviar 
o webhook (idempotência quebrada).

## 2. Problema 2 — Gateway só aceita form-urlencoded (ALTA)

Instância 151 envia webhooks com `Content-Type: application/json`. 
O `r.ParseForm()` não processa JSON → `data[]` vazio → erro 400.
Webhook é reenviado infinitamente pelo Iugu (flood no log).

## 3. Correções

### 3.1 `internal/pagamento/processar.go`
- Adicionar parâmetro `event string` a `ProcessarPagamento()`
- Inserir linha em `gisp_iugu_gatilhos` antes da TX:
  ```go
  db.Exec(`INSERT INTO gisp_iugu_gatilhos 
      (id, account_id, external_reference, status, event, dados_json, datetime_received)
      VALUES (?, ?, ?, ?, ?, ?, ?)
      ON DUPLICATE KEY UPDATE id = id`, ...)
  ```

### 3.2 `internal/worker/processar_pagamento_iugu.go`
- Passar `msg.Event` para `ProcessarPagamento()`

### 3.3 `internal/gateway/iugu_webhook.go`
- Adicionar fallback JSON quando `data[]` vazio:
  ```go
  if len(data) == 0 {
      body, _ := io.ReadAll(r.Body)
      var j struct { Event string; Data map[string]string `json:"data"` }
      if json.Unmarshal(body, &j) == nil && j.Event != "" {
          event = j.Event; data = j.Data
      }
  }
  ```

## 4. Impacto
- `gisp_iugu_gatilhos` agora é populado corretamente
- Idempotência restaurada (pagamento único garantido)
- Gateway aceita JSON + form-urlencoded
- Instância 151 para de floodar