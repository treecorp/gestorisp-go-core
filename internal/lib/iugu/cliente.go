package iugu

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FaturaIugu representa a resposta da API Iugu para uma fatura.
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
	Valor          string `json:"valor"`
	ValorPago      string `json:"valor_pago"`
	DataVencimento string `json:"data_vencimento"`
	DataPagamento  string `json:"data_pagamento"`
}

// ClienteIugu é o cliente HTTP para a API de cobrança Iugu.
type ClienteIugu struct {
	apiURL string
	apiKey string
	http   *http.Client
}

// NovoClienteIugu cria e retorna um ClienteIugu configurado com a
// URL base e chave de API fornecidas.
func NovoClienteIugu(apiURL, apiKey string) *ClienteIugu {
	return &ClienteIugu{
		apiURL: apiURL,
		apiKey: apiKey,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ConsultarFatura busca os dados de uma fatura na API Iugu pelo seu
// identificador. Retorna erro se a fatura não existir ou a API falhar.
func ConsultarFatura(cliente *ClienteIugu, faturaID string) (*FaturaIugu, error) {
	url := fmt.Sprintf("%s/faturas/%s", cliente.apiURL, faturaID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.SetBasicAuth(cliente.apiKey, "")
	req.Header.Set("Accept", "application/json")

	resp, err := cliente.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("iugu: erro ao consultar API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("iugu: erro ao ler resposta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iugu: status %d, body: %s", resp.StatusCode, string(body))
	}

	var fatura FaturaIugu
	if err := json.Unmarshal(body, &fatura); err != nil {
		return nil, fmt.Errorf("iugu: erro ao decodificar resposta: %w", err)
	}

	if fatura.Status == "" {
		return nil, fmt.Errorf("iugu: resposta invalida: status vazio")
	}

	return &fatura, nil
}
