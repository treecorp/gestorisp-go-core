// Package pagamento implementa as regras de negócio para processamento
// de pagamentos Iugu, incluindo baixa contábil, desbloqueio de contratos,
// lançamento em caixa e geração de protocolos.
//
// Este pacote foi extraído de internal/pagamento/processar.go e utiliza
// interfaces de repositório para permitir testes sem banco de dados real.
// Nenhuma função neste pacote executa SQL diretamente — toda persistência
// é delegada às interfaces definidas em service.go.
//
// SDD-022: Service de Pagamento
package pagamento
