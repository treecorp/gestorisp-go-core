package pagamento

// codigosOrigem mapeia o método de pagamento retornado pela API Iugu para
// o código de origem utilizado no lançamento contábil do caixa.
//
// Valores possíveis:
//   - iugu_pix / iugu_pix_test → "5" (PIX)
//   - iugu_credit_card          → "4" (Cartão de Crédito)
//   - iugu_bank_slip / iugu_bank_slip_test → "7" (Boleto Bancário)
//   - demais métodos            → "7" (Boleto, fallback)
var codigosOrigem = map[string]string{
	"iugu_pix":            "5",
	"iugu_pix_test":       "5",
	"iugu_credit_card":    "4",
	"iugu_bank_slip":      "7",
	"iugu_bank_slip_test": "7",
}

// OrigemPagamento retorna o código de origem contábil baseado no método
// de pagamento informado pela API Iugu. Se o método não estiver mapeado,
// retorna "7" (boleto bancário como fallback).
func OrigemPagamento(metodo string) string {
	if cod, ok := codigosOrigem[metodo]; ok {
		return cod
	}
	return "7"
}
