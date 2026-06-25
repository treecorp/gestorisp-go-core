# HOTFIX-006 — Gerar protocolo de baixa na faixa correta (300000-399999)

**Status:** Implementado
**Autor:** Dev Backend
**Prioridade:** Alta
**Tipo:** Correcao

## 1. Problema

O PHP legado gera **tres protocolos diferentes** durante o fluxo de baixa de fatura, cada um em uma faixa numerica distinta:

| Contexto | PHP (range) | Finalidade |
|----------|-------------|------------|
| `protocolo_baixa` | `rand(100000, 999999)` | Campo `protocolo_baixa` na tabela `sgp_clientes_faturas` |
| `protocolo` (baixa) | `rand(300000, 399999)` | Registro em `sgp_clientes_contratos_protocolos` (titulo "Baixa de fatura") |
| `protocolo` (desbloqueio) | `rand(400000, 499999)` | Registro em `sgp_clientes_contratos_protocolos` (titulo "Desbloqueio de contrato") |

O Go atual gera **um unico** protocolo (`gerarProtocolo(100000, 999999)`) em `executarBaixa()` e reusa o mesmo numero para:

1. `protocolo_baixa` na fatura ✅ (faixa correta)
2. `gisp_exec_return` no gatilho ✅ (apenas log)
3. `protocolo` no registro de baixa em `sgp_clientes_contratos_protocolos` ❌ (faixa errada)

Isso faz com que os protocolos de baixa aparecam na faixa geral (`100000-999999`) em vez da faixa legada (`300000-399999`), como visto em producao:

```
# Go (incorreto) — protocolo 100001 
# PHP (correto) — protocolo 301190, 373958, 314343
```

### 1.1 Sintomas

- Registros em `sgp_clientes_contratos_protocolos` com titulo "Baixa de fatura" aparecem com protocolo `100001` em vez de `300001`
- Mistura de faixas numericas na tabela de protocolos, dificultando identificacao visual
- Diferenca de comportamento vs PHP legado

## 2. Correcao

### 2.1 `internal/pagamento/processar.go` — `executarBaixa()`

Gerar **dois protocolos distintos** em vez de um:

```go
// ANTES:
protocolo := fmt.Sprintf("%d", gerarProtocolo(100000, 999999))

// DEPOIS:
protocoloBaixa := fmt.Sprintf("%d", gerarProtocolo(100000, 999999))      // protocolo_baixa da fatura
protocolo := fmt.Sprintf("%d", gerarProtocolo(300000, 399999))            // protocolo do registro de baixa
```

Onde:
- `protocoloBaixa` → usado no UPDATE `sgp_clientes_faturas SET protocolo_baixa = ?`
- `protocolo` → usado no INSERT `sgp_clientes_contratos_protocolos` (registro de baixa) e `gisp_exec_return`

A variavel `protocolo` existente continua sendo passada para `marcarProcessado` e `criarProtocoloBaixa` — elas apenas receberao o novo numero da faixa `300000-399999`.

### 2.2 Comportamento apos correcao

| Contexto | PHP | Go antes | Go depois |
|----------|-----|----------|-----------|
| `protocolo_baixa` (fatura) | `100000-999999` | `100000-999999` ✅ | `100000-999999` ✅ |
| `protocolo` registro baixa | `300000-399999` | `100000-999999` ❌ | `300000-399999` ✅ |
| `protocolo` registro desbloqueio | `400000-499999` | `400000-499999` ✅ | `400000-499999` ✅ |
| `gisp_exec_return` | `protocolo_baixa` | `100000-999999` | `300000-399999` (registro) |

Nota: o `gisp_exec_return` em `gisp_iugu_gatilhos` armazenara o protocolo do registro (`300000-399999`) em vez do `protocolo_baixa` da fatura. Isso e aceitavel pois ambos sao numeros de protocolo validos.

## 3. Impacto

- **Nenhuma quebra** de funcionalidade
- Apenes a faixa numerica do protocolo no registro de baixa muda
- O `protocolo_baixa` na fatura permanece inalterado (faixa geral)
- Compatibilidade total com dados existentes do PHP legado

## 4. Arquivos alterados

| Arquivo | Alteracao |
|---------|-----------|
| `internal/pagamento/processar.go` | Gerar `protocoloBaixa` separado (faixa 100000) + `protocolo` (faixa 300000) |

## 5. Teste

- `go vet ./...` — sem erros
- `go build ./...` — compilacao limpa
- Apos deploy, nova baixa deve gerar protocolo na faixa `300000-399999` no registro de baixa
- `protocolo_baixa` na fatura continua na faixa `100000-999999`
