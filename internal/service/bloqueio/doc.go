// Package bloqueio implementa as regras de negócio para bloqueio de
// clientes com faturas vencidas, incluindo lógica de decisão, aplicação
// do bloqueio em banco de dados e cálculo de dias em atraso.
//
// Este pacote foi extraído de internal/worker/listar_clientes_vencidos.go
// e utiliza interfaces de repositório para permitir testes sem banco de
// dados real. Nenhuma função neste pacote executa SQL diretamente — toda
// persistência é delegada às interfaces definidas em service.go.
//
// SDD-023: Service de Bloqueio
package bloqueio
