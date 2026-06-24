# Decisao: Port do cron_1 com go-routeros (RouterOS API) + fuso centralizado

## Contexto
O cron_1 e o worker mais critico do sistema. Ele sincroniza sessoes RADIUS,
corrige status Online/Offline, desbloqueia usuarios travados (via RouterBoard),
e repara inconsistencias de conexao.

## Decisao
1. **Reimplementar toda logica do ws-rabbimq PHP em Go puro**
   - Elimina chamada HTTPS para o Cloud Run PHP
   - Worker Go acessa banco da instancia diretamente
   - 5 sub-rotinas: sync_conexoes_radius, sync_conexoes_radius_status,
     desbloqueia_user_bloqueado_no_banco, reparar_offline_para_online,
     reparar_online_para_offline

2. **Usar `github.com/go-routeros/routeros/v3` para comunicacao RouterBoard**
   - Substitui a classe PHP `RouterosAPI` (socket TCP raw)
   - Dial com timeout de 5s / usuario nao ativo → preserva sessao
   - FUTURO: mover verificacao RouterOS para o WS2

3. **Fuso centralizado via `internal/infra/fuso`**
   - `fuso.Agora()` em vez de `time.Now()` em todo o codigo
   - Container precisa de `TZ=America/Sao_Paulo` para timestamps 100% compativeis com PHP
   - Se MySQL estiver em timezone diferente, nao importa — nao confiamos em `NOW()` do MySQL
   - Apenas uma variavel (`fuso.LocalBR`) para mudar o timezone do sistema inteiro

7. **Expressao cron:** `0 */5 0,3-23 * * *` (5 em 5 min, exceto 01-02h)

## Consequencias
- Worker Go processa localmente sem depender de Cloud Run PHP
- Criptografia CI3 nao necessaria (cron_1 nunca usou)
- Uma dependencia externa: go-routeros/v3
- Container precisa ter `TZ=America/Sao_Paulo`

## Arquivos criados/modificados
- `internal/worker/cron_1.go` — handler + 5 sub-rotinas (~500 linhas)
- `internal/infra/fuso/fuso.go` — fuso centralizado
- `internal/infra/routeros/client.go` — cliente RouterOS API
- `internal/worker/check_pop_status.go` — corrigido para usar fuso.Agora()
- `cmd/worker/main.go` — registrado consumidor cron_1
- `cmd/gestor/main.go` — expressao cron alterada para */5 min
