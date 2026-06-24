package pagamento

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type FaturaIugu struct {
	ID             string `json:"id"`
	AccountID      string `json:"account_id"`
	ExternalRef    string `json:"external_reference"`
	Status         string `json:"status"`
	TotalCents     int    `json:"total_cents"`
	TotalPaidCents int    `json:"total_paid_cents"`
	TaxesPaidCents int    `json:"taxes_paid_cents"`
	PaidAt         string `json:"paid_at"`
	PaymentMethod  string `json:"payment_method"`
	PayerName      string `json:"payer_name"`
	PayerCpfCnpj   string `json:"payer_cpf_cnpj"`
}

type ClienteIugu struct {
	token  string
	client *http.Client
}

func NovoCliente(token string) *ClienteIugu {
	return &ClienteIugu{
		token: token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *ClienteIugu) ConsultarFatura(faturaID string) (FaturaIugu, error) {
	url := fmt.Sprintf("https://api.iugu.com/v1/invoices/%s", faturaID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return FaturaIugu{}, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.SetBasicAuth(c.token, "")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return FaturaIugu{}, fmt.Errorf("erro ao consultar Iugu API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FaturaIugu{}, fmt.Errorf("erro ao ler resposta Iugu: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return FaturaIugu{}, fmt.Errorf("iugu api: status %d, body: %s", resp.StatusCode, string(body))
	}

	var fatura FaturaIugu
	if err := json.Unmarshal(body, &fatura); err != nil {
		return FaturaIugu{}, fmt.Errorf("erro ao decodificar resposta Iugu: %w", err)
	}

	if fatura.Status == "" {
		return FaturaIugu{}, fmt.Errorf("resposta Iugu invalida: status vazio")
	}

	return fatura, nil
}
