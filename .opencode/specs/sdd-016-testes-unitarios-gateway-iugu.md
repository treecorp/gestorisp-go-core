# SDD-016 — Testes Unitarios do Gateway Iugu

**Status:** Concluido
**Autor:** Dev Backend
**Prioridade:** Media
**Tipo:** Documentacao

## 1. Objetivo

Garantir que o parse de webhooks Iugu (form-urlencoded e JSON) e a logica de processamento de pagamento funcionem corretamente com dados reais antes do deploy em producao.

## 2. Arquivos de Teste

| Arquivo | Pacote | Testes | Dependencia Externa |
|---|---|---|---|
| `internal/gateway/iugu_webhook_test.go` | `gateway` | 7 | Nenhuma |
| `internal/pagamento/processar_test.go` | `pagamento` | 8 | Nenhuma |

### 2.1 Testes do Gateway (`internal/gateway/iugu_webhook_test.go`)

Usa payloads reais extraidos do banco `gisp_isp_local_1` (tabela `gisp_iugu_gatilhos`):

- **Igor Ricardo (PIX)**: `id=899E76B431864AAB854977026372A704`, `external_reference=6a3374c...`, `paid_cents=5000`
- **Daniela (PIX)**: `id=54660564FE2747E2BE45F6F6CE831A79`, `paid_cents=5104`, com `pix_end_to_end_id`
- **Ana Luiza (Boleto)**: `id=56112173A2AC41748841334D088720B1`, `payment_method=iugu_bank_slip`

| Teste | O que cobre |
|---|---|
| `TestParseFormData` | Parse de `POST` form-urlencoded com PHP-style `data[chave]` |
| `TestParseJSONData` | Parse de JSON `{"event":"...","data":{...}}` |
| `TestFormEJSONProduzemMesmoData` | Garante que form e JSON produzem o mesmo `map[string]string` |
| `TestParseFormPIX` | Parse de pagamento PIX com `pix_end_to_end_id` |
| `TestCamposEssenciaisPresentes` | Verifica que `id`, `external_reference`, `status`, `payment_method`, `paid_cents` estao sempre presentes |
| `TestMensagemPagamentoIuguSerializacao` | Serializacao/deserializacao da mensagem que trafega na fila RabbitMQ |
| `TestURLEncodingDados` | Validacao do URL encoding (`%5B` = `[`, `%5D` = `]`) |

### 2.2 Testes do Processamento (`internal/pagamento/processar_test.go`)

Testa todas as funcoes auxiliares e o mapeamento de dados reais:

| Teste | O que cobre |
|---|---|
| `TestCodigosOrigem` | Mapeamento `iugu_pix→5`, `iugu_credit_card→4`, `iugu_bank_slip→7`, fallback `desconhecido→7` |
| `TestLimparNumero` | `limparNumero("R$ 5.000,00") → "500000"`, etc. |
| `TestFormatarMoeda` | `formatarMoeda("5000") → "50,00"`, `formatarMoeda("5") → "0,05"` (padding zero) |
| `TestExternalRef` | `externalRef(Fatura{ExternalRef:"..."})` com valor preenchido e vazio |
| `TestMapDadosJSONIgorPIX` | Dados reais do Igor: `status=paid`, `origem=5`, `moeda=50,00` |
| `TestMapDadosJSONAnaLuizaBoleto` | Dados reais da Ana: `origem=7` (boleto), `moeda=50,00` |
| `TestMapDadosJSONDanielaPIX` | Dados reais da Daniela: `pix_end_to_end_id`, `origem=5`, `moeda=51,04` |
| `TestSerializacaoMensagemPagamento` | Serializacao completa da mensagem da fila com `Instancia`, `Event`, `Data`, `Tentativa` |

## 3. Como Executar

```bash
# Todos os testes (gateway + pagamento + dominio + worker)
go test -v ./internal/gateway/ ./internal/pagamento/ ./internal/dominio/ ./internal/worker/

# Apenas gateway
go test -v ./internal/gateway/

# Apenas pagamento
go test -v ./internal/pagamento/

# Com coverage
go test -cover ./internal/gateway/ ./internal/pagamento/
```

## 4. Dados de Teste

Os payloads foram extraidos do banco de producao `gisp_isp_local_1`:

```sql
SELECT id, external_reference, status, payment_method, paid_cents, payer_cpf_cnpj, payer_name, pix_end_to_end_id
FROM gisp_iugu_gatilhos
WHERE datetime_received >= '2026-03-01'
ORDER BY datetime_received DESC
LIMIT 10;
```

Nenhum dado sensivel real foi commitado — apenas `payer_name`, `payer_cpf_cnpj`, `paid_cents`, etc. que sao dados publicos de fatura.

## 5. Bug Encontrado e Corrigido

**`formatarMoeda`** retornava formato invalido para valores com 1 ou 2 digitos:

| Entrada | Antes | Depois |
|---|---|---|
| `"5"` (5 centavos) | `"0,5"` | `"0,05"` |
| `"0"` (0 centavos) | `"0,0"` | `"0,00"` |

**Causa:** `if len(limpo) <= 2 { return "0," + limpo }` nao fazia padding dos centavos para 2 digitos.

**Correcao:** Loop `for len(limpo) < 3 { limpo = "0" + limpo }` garante pelo menos 3 caracteres antes de separar reais/centavos.

## 6. Cobertura

16 testes no total, todos passando (`go vet` + `go test` limpos). Nenhuma dependencia externa (banco, RabbitMQ, RouterOS) nos testes unitarios — apenas o `internal/worker/desconectar_contrato_test.go` requer RouterOS real.
