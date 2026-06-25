package pagamento

import (
	"encoding/json"
	"testing"
)

func TestCodigosOrigem(t *testing.T) {
	tests := []struct {
		metodo string
		expected string
	}{
		{"iugu_pix", "5"},
		{"iugu_pix_test", "5"},
		{"iugu_credit_card", "4"},
		{"iugu_bank_slip", "7"},
		{"iugu_bank_slip_test", "7"},
		{"metodo_desconhecido", "7"},
		{"", "7"},
	}

	for _, tt := range tests {
		got := origemPagamento(tt.metodo)
		if got != tt.expected {
			t.Errorf("origemPagamento(%q) = %s, esperado %s", tt.metodo, got, tt.expected)
		}
	}
}

func TestLimparNumero(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5000", "5000"},
		{"5.000", "5000"},
		{"5,00", "500"},
		{"R$ 5.000,00", "500000"},
		{"1.234,56", "123456"},
		{"0", "0"},
		{"", ""},
	}

	for _, tt := range tests {
		got := limparNumero(tt.input)
		if got != tt.expected {
			t.Errorf("limparNumero(%q) = %q, esperado %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatarMoeda(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5000", "50,00"},
		{"100", "1,00"},
		{"5", "0,05"},
		{"123456", "1234,56"},
		{"0", "0,00"},
	}

	for _, tt := range tests {
		got := formatarMoeda(tt.input)
		if got != tt.expected {
			t.Errorf("formatarMoeda(%q) = %q, esperado %q", tt.input, got, tt.expected)
		}
	}
}

func TestExternalRef(t *testing.T) {
	tests := []struct {
		fatura   FaturaIugu
		expected string
	}{
		{FaturaIugu{ExternalRef: "6a3374c549e6f98e2c83894aab3f3f33ae33c1d3"}, "6a3374c549e6f98e2c83894aab3f3f33ae33c1d3"},
		{FaturaIugu{ExternalRef: ""}, ""},
	}

	for _, tt := range tests {
		got := externalRef(tt.fatura)
		if got != tt.expected {
			t.Errorf("externalRef(%+v) = %s, esperado %s", tt.fatura, got, tt.expected)
		}
	}
}

// Testa que o dados_json real da Iugu pode ser mapeado corretamente
// Dados extraidos do banco gisp_isp_local_1 (tabela gisp_iugu_gatilhos)
func TestMapDadosJSONIgorPIX(t *testing.T) {
	jsonStr := `{
		"id": "899E76B431864AAB854977026372A704",
		"account_id": "42591B1C08134F03B544FCE71238B582",
		"external_reference": "6a3374c549e6f98e2c83894aab3f3f33ae33c1d3",
		"paid_cents": "5000",
		"status": "paid",
		"payment_method": "iugu_pix",
		"payer_cpf_cnpj": "04720186475",
		"payer_name": "IGOR RICARDO LISBOA DE OLIVEIRA",
		"paid_at": "2026-06-22T18:32:40.000Z"
	}`

	var data map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("erro ao parsear JSON: %v", err)
	}

	if data["id"] != "899E76B431864AAB854977026372A704" {
		t.Errorf("id esperado 899E76..., got=%s", data["id"])
	}
	if data["external_reference"] != "6a3374c549e6f98e2c83894aab3f3f33ae33c1d3" {
		t.Errorf("external_reference incorreto")
	}
	if data["status"] != "paid" {
		t.Errorf("status esperado=paid, got=%s", data["status"])
	}
	if data["payment_method"] != "iugu_pix" {
		t.Errorf("payment_method esperado=iugu_pix, got=%s", data["payment_method"])
	}
	if data["paid_cents"] != "5000" {
		t.Errorf("paid_cents esperado=5000, got=%s", data["paid_cents"])
	}

	// Verificar mapeamento de origem
	origem := origemPagamento(data["payment_method"])
	if origem != "5" {
		t.Errorf("origem para iugu_pix esperado=5, got=%s", origem)
	}

	// Verificar centavos
	valor := limparNumero(data["paid_cents"])
	if valor != "5000" {
		t.Errorf("paid_cents limpo esperado=5000, got=%s", valor)
	}

	moeda := formatarMoeda(data["paid_cents"])
	if moeda != "50,00" {
		t.Errorf("paid_cents formatado esperado=50,00, got=%s", moeda)
	}
}

// Testa dados da Ana Luiza (boleto) - paga 5000 centavos
func TestMapDadosJSONAnaLuizaBoleto(t *testing.T) {
	jsonStr := `{
		"id": "56112173A2AC41748841334D088720B1",
		"account_id": "42591B1C08134F03B544FCE71238B582",
		"external_reference": "6a337505cad0e8a4a6ca62456f5b6d16b3bb7ece",
		"paid_cents": "5000",
		"status": "paid",
		"payment_method": "iugu_bank_slip",
		"payer_cpf_cnpj": "11675337470",
		"payer_name": "Ana Luiza Caula Cartaxo",
		"paid_at": "2026-06-22T19:12:17.000Z"
	}`

	var data map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("erro ao parsear JSON: %v", err)
	}

	if data["external_reference"] != "6a337505cad0e8a4a6ca62456f5b6d16b3bb7ece" {
		t.Errorf("external_reference incorreto")
	}

	origem := origemPagamento(data["payment_method"])
	if origem != "7" {
		t.Errorf("origem para iugu_bank_slip esperado=7, got=%s", origem)
	}

	moeda := formatarMoeda(data["paid_cents"])
	if moeda != "50,00" {
		t.Errorf("paid_cents formatado esperado=50,00, got=%s", moeda)
	}
}

// Testa dados da Daniela (PIX com pix_end_to_end_id)
func TestMapDadosJSONDanielaPIX(t *testing.T) {
	jsonStr := `{
		"id": "54660564FE2747E2BE45F6F6CE831A79",
		"account_id": "42591B1C08134F03B544FCE71238B582",
		"external_reference": "6a169c7be23adc374a447a3ce6ae3fffa282a0b6",
		"paid_cents": "5104",
		"status": "paid",
		"payment_method": "iugu_pix",
		"payer_cpf_cnpj": "05857303427",
		"payer_name": "Daniela Silva de Amorim",
		"pix_end_to_end_id": "E18236120202606241308s0474409e05",
		"paid_at": "2026-06-24T13:09:07.000Z"
	}`

	var data map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("erro ao parsear JSON: %v", err)
	}

	if data["pix_end_to_end_id"] != "E18236120202606241308s0474409e05" {
		t.Errorf("pix_end_to_end_id incorreto")
	}

	origem := origemPagamento(data["payment_method"])
	if origem != "5" {
		t.Errorf("origem para iugu_pix esperado=5, got=%s", origem)
	}

	moeda := formatarMoeda(data["paid_cents"])
	if moeda != "51,04" {
		t.Errorf("paid_cents formatado (5104) esperado=51,04, got=%s", moeda)
	}
}

// Testa a serializacao da MensagemPagamentoIugu (como se fosse publicar na fila)
func TestSerializacaoMensagemPagamento(t *testing.T) {
	dados := map[string]string{
		"id":                 "899E76B431864AAB854977026372A704",
		"account_id":         "42591B1C08134F03B544FCE71238B582",
		"external_reference": "6a3374c549e6f98e2c83894aab3f3f33ae33c1d3",
		"paid_cents":         "5000",
		"status":             "paid",
		"payment_method":     "iugu_pix",
		"payer_name":         "IGOR RICARDO LISBOA DE OLIVEIRA",
		"paid_at":            "2026-06-22T18:32:40.000Z",
	}

	type MensagemPagamentoIugu struct {
		Instancia map[string]interface{} `json:"instancia"`
		Event     string                 `json:"event"`
		Data      map[string]string      `json:"data"`
		Tentativa int                    `json:"tentativa"`
	}

	msg := MensagemPagamentoIugu{
		Instancia: map[string]interface{}{
			"id":       1,
			"token":    "teste-unitario",
			"hostname": "dbhost",
			"port":     "3306",
			"username": "user",
			"password": "pass",
			"database": "gisp_isp_local_1",
		},
		Event:     "invoice.status_changed",
		Data:      dados,
		Tentativa: 0,
	}

	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("erro ao serializar: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("erro ao decodificar: %v", err)
	}

	event, _ := decoded["event"].(string)
	if event != "invoice.status_changed" {
		t.Errorf("event esperado=invoice.status_changed, got=%s", event)
	}

	tentativa, _ := decoded["tentativa"].(float64)
	if tentativa != 0 {
		t.Errorf("tentativa esperado=0, got=%f", tentativa)
	}

	dataDecoded, ok := decoded["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data nao e um map")
	}
	if dataDecoded["external_reference"] != "6a3374c549e6f98e2c83894aab3f3f33ae33c1d3" {
		t.Errorf("external_reference incorreto")
	}
}
