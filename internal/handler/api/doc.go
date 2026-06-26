// Package api implementa o handler HTTP da API REST unificada (porta 8081).
//
// Expõe endpoints para desconexão PPPoE em RouterOS e webhooks de pagamento
// Iugu. O handler faz parsing das requisições, valida dados e publica na fila
// RabbitMQ para processamento assíncrono pelos workers.
package api
