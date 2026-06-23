package mensageria

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
)

// RabbitMQ gerencia a conexao e publicacao de mensagens no RabbitMQ
type RabbitMQ struct {
	cfg    config.RabbitMQConfig
	conn   *amqp.Connection
	canal  *amqp.Channel
	mu     sync.Mutex
	fechou bool
}

// Conectar estabelece conexao com o RabbitMQ
func Conectar(cfg config.RabbitMQConfig) (*RabbitMQ, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.Usuario, cfg.Senha, cfg.Host, cfg.Porta)
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no RabbitMQ: %w", err)
	}
	canal, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("falha ao abrir canal RabbitMQ: %w", err)
	}

	r := &RabbitMQ{cfg: cfg, conn: conn, canal: canal}
	r.iniciarMonitoramento()
	logger.Sucesso("rabbit", "Conectado ao RabbitMQ (%s:%s)", cfg.Host, cfg.Porta)
	return r, nil
}

// ConectarComRetry tenta conectar ao RabbitMQ em loop infinito com backoff progressivo.
// So retorna quando a conexao for estabelecida com sucesso.
// O backoff comeca em 2s e dobra ate o maximo de 60s entre tentativas.
func ConectarComRetry(cfg config.RabbitMQConfig) *RabbitMQ {
	espera := 2 * time.Second
	for {
		rabbit, err := Conectar(cfg)
		if err == nil {
			return rabbit
		}
		logger.Aviso("rabbit", "Falha ao conectar: %v. Reintentando em %s...", err, espera)
		time.Sleep(espera)
		espera *= 2
		if espera > 60*time.Second {
			espera = 60 * time.Second
		}
	}
}

// iniciarMonitoramento escuta notificacoes de fechamento da conexao
// e tenta reconectar automaticamente em loop infinito
func (r *RabbitMQ) iniciarMonitoramento() {
	go func() {
		notify := r.conn.NotifyClose(make(chan *amqp.Error))
		for err := range notify {
			r.mu.Lock()
			if r.fechou {
				r.mu.Unlock()
				return
			}
			r.mu.Unlock()

			logger.Aviso("rabbit", "Conexao perdida: %v. Reconectando...", err)
			r.reconectar()
			logger.Sucesso("rabbit", "Reconectado com sucesso")
		}
	}()
}

// reconectar tenta restabelecer a conexao com o RabbitMQ em loop infinito
func (r *RabbitMQ) reconectar() {
	r.mu.Lock()
	if r.canal != nil {
		r.canal.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
	r.mu.Unlock()

	espera := 2 * time.Second
	for {
		url := fmt.Sprintf("amqp://%s:%s@%s:%s/", r.cfg.Usuario, r.cfg.Senha, r.cfg.Host, r.cfg.Porta)
		conn, err := amqp.Dial(url)
		if err == nil {
			canal, err := conn.Channel()
			if err == nil {
				r.mu.Lock()
				r.conn = conn
				r.canal = canal
				r.mu.Unlock()
				r.iniciarMonitoramento()
				return
			}
			conn.Close()
		}
		logger.Aviso("rabbit", "Reconexao falhou: %v. Reintentando em %s...", err, espera)
		time.Sleep(espera)
		espera *= 2
		if espera > 60*time.Second {
			espera = 60 * time.Second
		}
	}
}

// Fechar encerra a conexao com o RabbitMQ
func (r *RabbitMQ) Fechar() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fechou = true
	if r.canal != nil {
		r.canal.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
	logger.Info("rabbit", "Conexao RabbitMQ encerrada")
}

// PublicarInstancia serializa a instancia como JSON -> Base64 e envia para a fila
// Mesmo formato usado pelo PHP original:
// $dados = base64_encode(json_encode($g));
// echo shell_exec("curl {$RABBITMQ_PRODUCER_HOST}/{fila}?dados={$dados}");
func (r *RabbitMQ) PublicarInstancia(fila string, instancia dominio.Instancia) error {
	payload := map[string]interface{}{
		"gisp_id":    instancia.ID,
		"gisp_token": instancia.Token,
		"hostname":   instancia.EnvDBHost,
		"username":   instancia.EnvDBUser,
		"password":   instancia.EnvDBPass,
		"database":   instancia.EnvDBName,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao serializar JSON: %w", err)
	}

	msg := base64.StdEncoding.EncodeToString(jsonBytes)

	r.mu.Lock()
	canal := r.canal
	r.mu.Unlock()

	if canal == nil {
		return fmt.Errorf("canal RabbitMQ nao disponivel")
	}

	_, err = canal.QueueDeclare(fila, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("erro ao declarar fila %s: %w", fila, err)
	}

	err = canal.Publish("", fila, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(msg),
	})
	if err != nil {
		return fmt.Errorf("erro ao publicar na fila %s: %w", fila, err)
	}

	return nil
}
