package cron

import (
	"github.com/robfig/cron/v3"

	"gestor/internal/config"
	"gestor/internal/infra/mensageria"
	"gestor/internal/infra/logger"
	"gestor/internal/cron/tarefas"
)

// TarefaRegistro define uma tarefa agendada com sua expressao cron e fila RabbitMQ
type TarefaRegistro struct {
	Expressao string // Expressao cron (com segundos)
	Nome      string // Nome amigavel para log
	Fila      string // Nome da fila RabbitMQ
}

// Agendador gerencia o agendamento e execucao das tarefas cron
type Agendador struct {
	cfg    *config.Config
	cron   *cron.Cron
	rabbit *mensageria.RabbitMQ
}

// NovoAgendador cria um novo agendador com as tarefas registradas
func NovoAgendador(cfg *config.Config, rabbit *mensageria.RabbitMQ) *Agendador {
	return &Agendador{
		cfg:    cfg,
		cron:   cron.New(cron.WithSeconds()),
		rabbit: rabbit,
	}
}

// Iniciar registra todas as tarefas e inicia o agendador
//
//	Todas as tarefas seguem o mesmo padrao:
//	1. Buscar instancias ativas no GISPADM
//	2. Para cada instancia, publicar na fila RabbitMQ correspondente
//
// Agendamentos (convertidos do start.sh original):
//
//	┌───────────── segundos (0-59)
//	│ ┌───────────── minutos (0-59)
//	│ │ ┌───────────── horas (0-23)
//	│ │ │ ┌───────────── dia do mes (1-31)
//	│ │ │ │ ┌───────────── mes (1-12)
//	│ │ │ │ │ ┌───────────── dia da semana (0-6)
//	│ │ │ │ │ │
//	* * * * * *
func (a *Agendador) Iniciar(tarefasRegistro []TarefaRegistro) {
	for _, t := range tarefasRegistro {
		tarefa := a.criarTarefa(t.Fila)
		id, err := a.cron.AddFunc(t.Expressao, tarefa)
		if err != nil {
			logger.Erro("agendador", "Erro ao registrar tarefa %s: %v", t.Nome, err)
			continue
		}
		logger.Sucesso("agendador", "Tarefa registrada: %s → expressao=%s → fila=%s",
			t.Nome, t.Expressao, t.Fila)
		_ = id
	}
	a.cron.Start()
	logger.Destaque("agendador", "Agendador iniciado com %d tarefas", len(tarefasRegistro))
}

// Parar para o agendador aguardando a conclusao das tarefas em execucao
func (a *Agendador) Parar() {
	ctx := a.cron.Stop()
	<-ctx.Done()
	logger.Info("agendador", "Agendador parado")
}

// criarTarefa retorna uma funcao que executa a logica para uma fila especifica
func (a *Agendador) criarTarefa(fila string) func() {
	return func() {
		tarefas.ExecutarParaTodasInstancias(a.cfg, a.rabbit, fila)
	}
}
