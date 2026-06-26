// Package repositorio implementa o padrão Repository para acesso a
// dados do sistema. Cada arquivo agrupa operações relacionadas a uma
// entidade (fatura, contrato, gatilho, bloqueio, etc.).
//
// Todas as funções recebem *sql.DB ou *sql.Tx explicitamente,
// permitindo controle transacional pelo chamador.
package repositorio
