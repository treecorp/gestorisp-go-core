package tarefas

import (
	"time"

	"gestor/internal/config"
	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/logger"
	"gestor/internal/infra/mensageria"
)

// ExecutarParaTodasInstancias le as instancias ativas do banco GISPADM
// e publica cada uma na fila RabbitMQ informada.
// Cada instancia e publicada individualmente com ate 3 tentativas.
// Se uma falhar, as demais continuam (nao bloqueia o lote).
//
// Reproduz o comportamento do PHP original:
//
//	$db = $this->get_cron();
//	foreach($db AS $g){
//	    $dados = base64_encode(json_encode($g));
//	    echo shell_exec("curl {$RABBITMQ_PRODUCER_HOST}/{fila}?dados={$dados}");
//	}
func ExecutarParaTodasInstancias(cfg *config.Config, rabbit *mensageria.RabbitMQ, fila string) {
	logger.Inicio(fila, "Iniciando execucao")

	instancias, err := banco.BuscarInstanciasAtivas(cfg.Banco, cfg)
	if err != nil {
		logger.Erro(fila, "Falha ao buscar instancias: %v", err)
		return
	}

	logger.Sucesso(fila, "%d instancias ativas encontradas", len(instancias))

	for _, instancia := range instancias {
		if err := publicarComRetry(rabbit, fila, instancia); err != nil {
			logger.Erro(fila, "Falha ao publicar instancia %d apos tentativas: %v", instancia.GetID(), err)
			continue
		}
		logger.Sucesso(fila, "Instancia %d publicada com sucesso", instancia.GetID())
	}

	logger.Sucesso(fila, "Concluido para %d instancias", len(instancias))
}

// publicarComRetry tenta publicar uma instancia na fila RabbitMQ com ate 3 tentativas
// e backoff curto entre elas (1s, 2s, 4s)
func publicarComRetry(rabbit *mensageria.RabbitMQ, fila string, instancia dominio.Instancia) error {
	tentativas := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	var ultimoErro error
	for i, espera := range tentativas {
		err := rabbit.PublicarInstancia(fila, instancia)
		if err == nil {
			return nil
		}
		ultimoErro = err
		if i < len(tentativas)-1 {
			logger.Aviso(fila, "Tentativa %d/3 falhou para instancia %d: %v. Reintentando em %s...",
				i+1, instancia.GetID(), err, espera)
			time.Sleep(espera)
		}
	}
	return ultimoErro
}
