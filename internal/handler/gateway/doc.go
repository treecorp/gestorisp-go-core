// Package gateway implementa o handler HTTP para webhooks de pagamento
// Iugu recebidos diretamente via porta de gateway (8082).
//
// Este handler faz parsing do webhook, autentica a instância pelo token
// e publica o evento na fila RabbitMQ para processamento assíncrono pelo
// worker. Nenhuma regra de negócio é executada aqui — apenas parsing HTTP
// e encaminhamento para a fila.
package gateway
