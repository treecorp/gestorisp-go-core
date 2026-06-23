
# Decisao 002: Retry infinito com backoff exponencial

**Data:** 23/06/2026
**Autor:** opencode
**Status:** ✅ Implementado

## Contexto

Inicialmente implementamos 5 tentativas de conexao com backoff. O usuario
testou e o banco estava inacessivel (rede sem VPN). O sistema tentou 5 vezes
e **morreu** com `exit status 1`.

O usuario exigiu: "nao quero que nunca ele tente parar de tentar para o
sistema nao cair".

## Decisao

Trocar de **5 tentativas com fallha fatal** para **loop infinito com backoff
progressivo**.

```
Antes: 5 tentativas -> log.Fatalf -> processo morre
Depois: infinito -> 2s -> 4s -> 8s -> ... -> 60s (cap) -> repete 60s
```

## Motivos

1. O sistema **nunca pode cair** — deve ficar tentando para sempre
2. O Docker reiniciaria o container, mas isso gera flapping e logs confusos
3. Melhor experiencia: um unico log a cada 60s "Aguardando conexao..."

## Implementacao

```go
func ConectarComRetry(cfg config.BancoConfig) *sql.DB {
    espera := 2 * time.Second
    for {
        db, err := Conectar(cfg)
        if err == nil {
            return db
        }
        logger.Aviso("banco", "Falha: %v. Reintentando em %s...", err, espera)
        time.Sleep(espera)
        espera *= 2
        if espera > 60*time.Second {
            espera = 60 * time.Second
        }
    }
}
```

## Impacto

- **Funcao mudou de assinatura:** antes retornava `(*sql.DB, error)`, agora
  retorna so `*sql.DB` (nunca falha)
- **main.go simplificado:** nao usa mais `os.Exit(1)`
- Reconexao durante execucao tambem usa loop infinito (NotifyClose + reconectar)

## Arquivos envolvidos
- `internal/infra/banco/mysql.go` — `ConectarComRetry`
- `internal/infra/mensageria/rabbit.go` — `ConectarComRetry` + `reconectar`
- `cmd/gestor/main.go` — removido `os.Exit`
