
# Padrao 001: Tarefa cron config-driven

**Data:** 23/06/2026
**Status:** ✅ Ativo

## Descricao

Todas as 7 tarefas cron seguem o **mesmo comportamento** — buscar instancias
ativas no GISPADM e publicar cada uma na fila RabbitMQ correspondente.

Ao inves de criar 7 funcoes separadas (cron_um.go, executar_cluster.go,
etc.), usamos uma **unica funcao generica** configurada por dados.

## Implementacao

```go
// main.go — configuracao das tarefas
agendador.Iniciar([]cron.TarefaRegistro{
  {Expressao: "0 * * * * *",     Nome: "cron_um",              Fila: "cron_1"},
  {Expressao: "0 */6 0,3-23 * * *", Nome: "executar_cluster",  Fila: "run_cluster"},
  {Expressao: "0 * * * * *",     Nome: "verificar_status_pop", Fila: "check_pop_status"},
  // ...
})
```

```go
// agendador.go — cria tarefa unica para cada registro
func (a *Agendador) criarTarefa(fila string) func() {
    return func() {
        tarefas.ExecutarParaTodasInstancias(a.cfg, a.rabbit, fila)
    }
}
```

## Quando usar

Sempre que uma nova fila precisar ser adicionada. Basta incluir uma nova
entrada no slice `TarefaRegistro` em `main.go`.

## Vantagens

- Zero duplicacao de codigo
- Adicionar nova tarefa = 1 linha no `main.go`
- Remover tarefa = so comentar/remover a linha
- Consistencia garantida entre tarefas
