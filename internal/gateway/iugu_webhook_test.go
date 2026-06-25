package gateway

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// dados_json real extraido do banco gisp_isp_local_1
// Fatura do Igor Ricardo (CPF 04720186475)
// id=899E76B431864AAB854977026372A704
// ext_ref=6a3374c549e6f98e2c83894aab3f3f33ae33c1d3
// event=invoice.status_changed
// status=paid

const payloadFormIgor = "event=invoice.status_changed" +
	"&data%5Bid%5D=899E76B431864AAB854977026372A704" +
	"&data%5Baccount_id%5D=42591B1C08134F03B544FCE71238B582" +
	"&data%5Bexternal_reference%5D=6a3374c549e6f98e2c83894aab3f3f33ae33c1d3" +
	"&data%5Bpaid_cents%5D=5000" +
	"&data%5Bstatus%5D=paid" +
	"&data%5Bpayment_method%5D=iugu_pix" +
	"&data%5Bpayer_cpf_cnpj%5D=04720186475" +
	"&data%5Bpayer_name%5D=IGOR+RICARDO+LISBOA+DE+OLIVEIRA" +
	"&data%5Bpaid_at%5D=2026-06-22T18%3A32%3A40.000Z"

const payloadJSONIgor = `{
	"event": "invoice.status_changed",
	"data": {
		"id": "899E76B431864AAB854977026372A704",
		"account_id": "42591B1C08134F03B544FCE71238B582",
		"external_reference": "6a3374c549e6f98e2c83894aab3f3f33ae33c1d3",
		"paid_cents": "5000",
		"status": "paid",
		"payment_method": "iugu_pix",
		"payer_cpf_cnpj": "04720186475",
		"payer_name": "IGOR RICARDO LISBOA DE OLIVEIRA",
		"paid_at": "2026-06-22T18:32:40.000Z"
	}
}`

// Dados da Daniela (PIX)
// id=54660564FE2747E2BE45F6F6CE831A79
// ext_ref=6a169c7be23adc374a447a3ce6ae3fffa282a0b6

const payloadFormDanielaPIX = "event=invoice.status_changed" +
	"&data%5Bid%5D=54660564FE2747E2BE45F6F6CE831A79" +
	"&data%5Baccount_id%5D=42591B1C08134F03B544FCE71238B582" +
	"&data%5Bexternal_reference%5D=6a169c7be23adc374a447a3ce6ae3fffa282a0b6" +
	"&data%5Bpaid_cents%5D=5104" +
	"&data%5Bstatus%5D=paid" +
	"&data%5Bpayment_method%5D=iugu_pix" +
	"&data%5Bpayer_cpf_cnpj%5D=05857303427" +
	"&data%5Bpayer_name%5D=Daniela+Silva+de+Amorim" +
	"&data%5Bpix_end_to_end_id%5D=E18236120202606241308s0474409e05" +
	"&data%5Bpaid_at%5D=2026-06-24T13%3A09%3A07.000Z"

// Dados da Ana Luiza (Boleto)
// id=56112173A2AC41748841334D088720B1
// ext_ref=6a337505cad0e8a4a6ca62456f5b6d16b3bb7ece

const payloadJSONAnaLuiza = `{
	"event": "invoice.status_changed",
	"data": {
		"id": "56112173A2AC41748841334D088720B1",
		"account_id": "42591B1C08134F03B544FCE71238B582",
		"external_reference": "6a337505cad0e8a4a6ca62456f5b6d16b3bb7ece",
		"paid_cents": "5000",
		"status": "paid",
		"payment_method": "iugu_bank_slip",
		"payer_cpf_cnpj": "11675337470",
		"payer_name": "Ana Luiza Caula Cartaxo",
		"paid_at": "2026-06-22T19:12:17.000Z"
	}
}`

func montarRequestForm(payload string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/pagamentos/iugu/gatilho/token", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func montarRequestJSON(payload string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/pagamentos/iugu/gatilho/token", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// Testa o parse do formulario urlencoded (PHP-style data[])
func TestParseFormData(t *testing.T) {
	req := montarRequestForm(payloadFormIgor)

	if err := req.ParseForm(); err != nil {
		t.Fatalf("erro ao parsear form: %v", err)
	}

	event := req.PostFormValue("event")
	if event != "invoice.status_changed" {
		t.Errorf("event esperado=invoice.status_changed, got=%s", event)
	}

	data := make(map[string]string)
	for key, values := range req.Form {
		if strings.HasPrefix(key, "data[") && strings.HasSuffix(key, "]") {
			campo := key[5 : len(key)-1]
			if len(values) > 0 {
				data[campo] = values[0]
			}
		}
	}

	expected := map[string]string{
		"id":                 "899E76B431864AAB854977026372A704",
		"account_id":         "42591B1C08134F03B544FCE71238B582",
		"external_reference": "6a3374c549e6f98e2c83894aab3f3f33ae33c1d3",
		"paid_cents":         "5000",
		"status":             "paid",
		"payment_method":     "iugu_pix",
		"payer_cpf_cnpj":     "04720186475",
		"payer_name":         "IGOR RICARDO LISBOA DE OLIVEIRA",
		"paid_at":            "2026-06-22T18:32:40.000Z",
	}

	for k, expectedV := range expected {
		got, ok := data[k]
		if !ok {
			t.Errorf("campo data[%s] nao encontrado", k)
			continue
		}
		if got != expectedV {
			t.Errorf("data[%s] esperado=%s, got=%s", k, expectedV, got)
		}
	}
}

// Testa o parse do JSON
func TestParseJSONData(t *testing.T) {
	var j struct {
		Event string            `json:"event"`
		Data  map[string]string `json:"data"`
	}

	if err := json.Unmarshal([]byte(payloadJSONIgor), &j); err != nil {
		t.Fatalf("erro ao parsear JSON: %v", err)
	}

	if j.Event != "invoice.status_changed" {
		t.Errorf("event esperado=invoice.status_changed, got=%s", j.Event)
	}

	expected := map[string]string{
		"id":                 "899E76B431864AAB854977026372A704",
		"account_id":         "42591B1C08134F03B544FCE71238B582",
		"external_reference": "6a3374c549e6f98e2c83894aab3f3f33ae33c1d3",
		"paid_cents":         "5000",
		"status":             "paid",
		"payment_method":     "iugu_pix",
		"payer_cpf_cnpj":     "04720186475",
		"payer_name":         "IGOR RICARDO LISBOA DE OLIVEIRA",
		"paid_at":            "2026-06-22T18:32:40.000Z",
	}

	for k, expectedV := range expected {
		got, ok := j.Data[k]
		if !ok {
			t.Errorf("data.%s nao encontrado", k)
			continue
		}
		if got != expectedV {
			t.Errorf("data.%s esperado=%s, got=%s", k, expectedV, got)
		}
	}
}

// Testa que form E JSON produzem o mesmo resultado
func TestFormEJSONProduzemMesmoData(t *testing.T) {
	// Form
	reqForm := montarRequestForm(payloadFormIgor)
	reqForm.ParseForm()
	dataForm := make(map[string]string)
	for key, values := range reqForm.Form {
		if strings.HasPrefix(key, "data[") && strings.HasSuffix(key, "]") {
			campo := key[5 : len(key)-1]
			if len(values) > 0 {
				dataForm[campo] = values[0]
			}
		}
	}

	// JSON
	var j struct {
		Event string            `json:"event"`
		Data  map[string]string `json:"data"`
	}
	json.Unmarshal([]byte(payloadJSONIgor), &j)

	if len(dataForm) != len(j.Data) {
		t.Errorf("form tem %d campos, json tem %d campos", len(dataForm), len(j.Data))
	}

	for k, vForm := range dataForm {
		vJSON, ok := j.Data[k]
		if !ok {
			t.Errorf("campo %s existe no form mas nao no json", k)
			continue
		}
		if vForm != vJSON {
			t.Errorf("campo %s: form=%s json=%s", k, vForm, vJSON)
		}
	}
}

// Testa o parse de PIX com pix_end_to_end_id
func TestParseFormPIX(t *testing.T) {
	req := montarRequestForm(payloadFormDanielaPIX)
	if err := req.ParseForm(); err != nil {
		t.Fatalf("erro ao parsear form: %v", err)
	}

	data := make(map[string]string)
	for key, values := range req.Form {
		if strings.HasPrefix(key, "data[") && strings.HasSuffix(key, "]") {
			campo := key[5 : len(key)-1]
			if len(values) > 0 {
				data[campo] = values[0]
			}
		}
	}

	if data["payment_method"] != "iugu_pix" {
		t.Errorf("payment_method esperado=iugu_pix, got=%s", data["payment_method"])
	}
	if data["pix_end_to_end_id"] == "" {
		t.Error("pix_end_to_end_id nao deveria estar vazio para pagamento PIX")
	}
	if data["payer_cpf_cnpj"] != "05857303427" {
		t.Errorf("payer_cpf_cnpj esperado=05857303427, got=%s", data["payer_cpf_cnpj"])
	}
}

// Testa que os campos essenciais estao sempre presentes
func TestCamposEssenciaisPresentes(t *testing.T) {
	payloads := []struct {
		nome    string
		jsonStr string
	}{
		{"Igor PIX", payloadJSONIgor},
		{"Ana Luiza Boleto", payloadJSONAnaLuiza},
	}

	for _, p := range payloads {
		t.Run(p.nome, func(t *testing.T) {
			var j struct {
				Event string            `json:"event"`
				Data  map[string]string `json:"data"`
			}
			if err := json.Unmarshal([]byte(p.jsonStr), &j); err != nil {
				t.Fatalf("erro JSON: %v", err)
			}

			if j.Data["id"] == "" {
				t.Error("id ausente")
			}
			if j.Data["external_reference"] == "" {
				t.Error("external_reference ausente")
			}
			if j.Data["status"] == "" {
				t.Error("status ausente")
			}
			if j.Data["payment_method"] == "" {
				t.Error("payment_method ausente")
			}
			if j.Data["paid_cents"] == "" {
				t.Error("paid_cents ausente")
			}
		})
	}
}

// Testa a serializacao da MensagemPagamentoIugu (mesmo formato da fila)
func TestMensagemPagamentoIuguSerializacao(t *testing.T) {
	var j struct {
		Event string            `json:"event"`
		Data  map[string]string `json:"data"`
	}
	json.Unmarshal([]byte(payloadJSONIgor), &j)

	msg := map[string]interface{}{
		"instancia": map[string]interface{}{
			"id":    1,
			"token": "teste",
		},
		"event": j.Event,
		"data":  j.Data,
	}

	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("erro ao serializar mensagem: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("erro ao decodificar: %v", err)
	}

	event, _ := decoded["event"].(string)
	if event != "invoice.status_changed" {
		t.Errorf("event esperado=invoice.status_changed, got=%s", event)
	}

	dataDecoded, ok := decoded["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data nao e um map")
	}
	if dataDecoded["external_reference"] != "6a3374c549e6f98e2c83894aab3f3f33ae33c1d3" {
		t.Errorf("external_reference incorreto")
	}
}

// Testa URL encoded para garantir que o encoding dos dados esta correto
func TestURLEncodingDados(t *testing.T) {
	raw := "data%5Bid%5D=899E76B431864AAB854977026372A704"
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		t.Fatalf("erro ao decodificar url: %v", err)
	}
	if decoded != "data[id]=899E76B431864AAB854977026372A704" {
		t.Errorf("decodificacao incorreta: %s", decoded)
	}
}
