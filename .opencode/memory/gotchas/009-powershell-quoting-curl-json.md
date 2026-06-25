# GOTCHA-009 — PowerShell quoting com curl.exe para JSON quebra payload

**Data:** 25/06/2026
**Contexto:** Teste do gateway Iugu em producao

## Problema

Ao chamar `curl.exe` do PowerShell com um JSON inline:

```powershell
curl.exe -X POST https://exemplo.com -H "Content-Type: application/json" -d '{"event":"test"}'
```

O PowerShell passa as aspas simples para o curl.exe, mas o parse interno do PowerShell pode quebrar o conteudo do JSON, resultando em payload corrompido. O gateway recebia um JSON invalido e retornava `400`.

## Sintomas

- Gateway log: `⚠ event nao informado` (mesmo enviando JSON com event)
- `curl -w "%{http_code}"` retorna `400`
- O mesmo comando funciona no Linux/Mac ou CMD.exe classico

## Causa

O PowerShell 5.1 tem comportamento imprevisivel ao passar argumentos com aspas duplas para comandos nativos (`curl.exe`). A string JSON pode ser reinterpretada, perdendo aspas internas.

## Solucao

Duas alternativas:

### 1. Usar arquivo temporario (recomendado)

```powershell
$tmp = [System.IO.Path]::GetTempFileName()
'{"event":"test"}' | Set-Content -Path $tmp -Encoding ASCII
curl.exe -s -d "@$tmp" ...
Remove-Item -Path $tmp
```

### 2. Usar `Invoke-RestMethod` (nativo PowerShell)

```powershell
$body = '{"event":"test"}'
Invoke-RestMethod -Uri "https://..." -Method Post -Body $body -ContentType "application/json"
```

## Impacto

Apenas durante testes manuais. Em producao, o gateway recebe webhooks do Iugu (form-urlencoded) ou do PHP (JSON via cURL do Linux), que nao tem esse problema.
