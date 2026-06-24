# Gotcha: WS2 retorna JSON array, nao objeto

## Contexto
O endpoint `/WS/routeros_system_health/{encrypted}` do WS2 retorna a saude do
RouterOS (temperatura, fans, voltagem, etc.).

## Sintoma
O worker `check_pop_status` tratava todo POP como offline, mesmo quando o
RouterOS estava acessivel. O `status_timeout` nunca era zerado.

## Causa
O WS2 retorna um **JSON array**:
```json
[{".id":"*11","name":"cpu-temperature","value":"38","type":"C"}, ...]
```

O codigo tentava fazer `json.Unmarshal(body, &map[string]interface{})` — que
espera um objeto `{}`. O parse falhava silenciosamente, e o worker caia no
`atualizarOffline`.

## CorrecAo
Trocar `map[string]interface{}` por `[]interface{}`:
```go
// antes
var resultado map[string]interface{}

// depois
var resultado []interface{}
```

`len(resultado) > 0` funciona para ambos: array vazio = offline,
array com itens = online.

## Referencia
- `internal/worker/check_pop_status.go` — linha 84

## Afeta
- Worker `check_pop_status` (fila `check_pop_status`)
