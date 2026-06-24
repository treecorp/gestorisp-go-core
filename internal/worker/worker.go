package worker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/streadway/amqp"

	"gestor/internal/config"
	"gestor/internal/dominio"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
	"gestor/internal/infra/observabilidade"
)

type Worker struct {
	cfg    *config.Config
	rabbit *mensageria.RabbitMQ
	conn   *amqp.Connection
	mu     sync.Mutex
}

func NovoWorker(cfg *config.Config, rabbit *mensageria.RabbitMQ) *Worker {
	return &Worker{
		cfg:    cfg,
		rabbit: rabbit,
	}
}

func (w *Worker) Iniciar(consumidores []Consumidor) {
	var wg sync.WaitGroup

	for _, c := range consumidores {
		wg.Add(1)
		go func(cons Consumidor) {
			defer wg.Done()
			w.consumir(cons)
		}(c)
	}

	wg.Wait()
}

func (w *Worker) consumir(cons Consumidor) {
	tag := cons.Fila
	logger.Inicio(tag, "Worker iniciando consumo da fila %s", cons.Fila)

	for {
		canal, err := w.obterCanal()
		if err != nil {
			logger.Erro(tag, "Falha ao obter canal: %v. Reintentando em 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		_, err = canal.QueueDeclare(cons.Fila, false, false, false, false, nil)
		if err != nil {
			logger.Erro(tag, "Falha ao declarar fila %s: %v. Reintentando em 5s...", cons.Fila, err)
			w.fecharCanal(canal)
			time.Sleep(5 * time.Second)
			continue
		}

		err = canal.Qos(1, 0, false)
		if err != nil {
			logger.Erro(tag, "Falha ao setar prefetch 1: %v. Reintentando em 5s...", err)
			w.fecharCanal(canal)
			time.Sleep(5 * time.Second)
			continue
		}

		mensagens, err := canal.Consume(cons.Fila, "", false, false, false, false, nil)
		if err != nil {
			logger.Erro(tag, "Falha ao consumir fila %s: %v. Reintentando em 5s...", cons.Fila, err)
			w.fecharCanal(canal)
			time.Sleep(5 * time.Second)
			continue
		}

		logger.Sucesso(tag, "Consumindo mensagens da fila %s...", cons.Fila)

		for msg := range mensagens {
			w.processarMensagem(tag, msg, cons.Handler)
		}

		logger.Aviso(tag, "Canal de mensagens fechado. Reintentando...")
		w.fecharCanal(canal)
		time.Sleep(2 * time.Second)
	}
}

func (w *Worker) fecharCanal(canal *amqp.Channel) {
	if canal != nil {
		canal.Close()
	}
}

func (w *Worker) processarMensagem(tag string, msg amqp.Delivery, handler func(dominio.Instancia) error) {
	instancia, err := decodificarMensagem(msg.Body)
	if err != nil {
		logger.Erro(tag, "Erro ao decodificar mensagem: %v", err)
		msg.Nack(false, false)
		return
	}

	logger.Info(tag, "Processando instancia %d (%s)", instancia.ID, instancia.EnvDBName)

	inicio := time.Now()
	err = handler(instancia)
	duracao := time.Since(inicio).Milliseconds()

	if err != nil {
		logger.Erro(tag, "Erro ao processar instancia %d: %v", instancia.ID, err)
		observabilidade.LogCompleto("erro", tag, "Instancia %d: %v em %dms", instancia.ID, duracao, instancia.ID, err, duracao)
		msg.Nack(false, false)
		return
	}

	logger.Sucesso(tag, "Instancia %d processada com sucesso", instancia.ID)
	observabilidade.LogCompleto("sucesso", tag, "Instancia %d processada em %dms", instancia.ID, duracao, instancia.ID, duracao)
	msg.Ack(false)
}

func decodificarMensagem(body []byte) (dominio.Instancia, error) {
	jsonBytes, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		return dominio.Instancia{}, fmt.Errorf("erro ao decodificar base64: %w", err)
	}

	var payload struct {
		GispID    int    `json:"gisp_id"`
		GispToken string `json:"gisp_token"`
		Hostname  string `json:"hostname"`
		Port      string `json:"port"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		Database  string `json:"database"`
	}

	if err := json.Unmarshal(jsonBytes, &payload); err != nil {
		return dominio.Instancia{}, fmt.Errorf("erro ao decodificar JSON: %w", err)
	}

	return dominio.Instancia{
		ID:        payload.GispID,
		Token:     payload.GispToken,
		EnvDBHost: payload.Hostname,
		EnvDBPort: payload.Port,
		EnvDBUser: payload.Username,
		EnvDBPass: payload.Password,
		EnvDBName: payload.Database,
	}, nil
}

func (w *Worker) obterCanal() (*amqp.Channel, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		canal, err := w.conn.Channel()
		if err == nil {
			return canal, nil
		}
		w.conn.Close()
		w.conn = nil
	}

	url := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		w.cfg.RabbitMQ.Usuario, w.cfg.RabbitMQ.Senha,
		w.cfg.RabbitMQ.Host, w.cfg.RabbitMQ.Porta,
	)
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no RabbitMQ: %w", err)
	}

	canal, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("falha ao abrir canal: %w", err)
	}

	w.conn = conn
	return canal, nil
}
