package gateway

import (
	"database/sql"
	"net/http"
	"strings"

	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/fuso"
	"gestor/internal/infra/logger"
)

var codigosOrigem = map[string]string{
	"iugu_pix":            "5",
	"iugu_pix_test":       "5",
	"iugu_credit_card":    "4",
	"iugu_bank_slip":      "7",
	"iugu_bank_slip_test": "7",
}

const tag = "gateway"

func HandleWebhook(w http.ResponseWriter, r *http.Request, instancia dominio.Instancia) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo nao permitido", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Aviso(tag, "Instancia %d: erro ao parsear form: %v", instancia.ID, err)
		http.Error(w, "Erro ao parsear form", http.StatusBadRequest)
		return
	}

	event := r.PostFormValue("event")
	if event == "" {
		logger.Aviso(tag, "Instancia %d: event nao informado no POST", instancia.ID)
		http.Error(w, "event nao informado", http.StatusBadRequest)
		return
	}

	data := make(map[string]string)
	for key, values := range r.Form {
		if strings.HasPrefix(key, "data[") && strings.HasSuffix(key, "]") {
			campo := key[5 : len(key)-1]
			if len(values) > 0 {
				data[campo] = values[0]
			}
		}
	}

	if len(data) == 0 {
		logger.Aviso(tag, "Instancia %d: dados data[] nao encontrados no POST", instancia.ID)
		http.Error(w, "data nao informado", http.StatusBadRequest)
		return
	}

	iuguID := data["id"]
	status := data["status"]
	externalRef := data["external_reference"]
	payerName := data["payer_name"]

	logger.Info(tag, "Webhook: instancia=%d event=%s iugu_fatura=%s status=%s ref=%s pagador=%s",
		instancia.ID, event, iuguID, status, truncate(externalRef, 20), payerName)

	db, err := banco.ConectarInstancia(
		instancia.EnvDBHost, instancia.EnvDBPort,
		instancia.EnvDBUser, instancia.EnvDBPass, instancia.EnvDBName,
	)
	if err != nil {
		logger.Erro(tag, "Instancia %d: erro ao conectar banco: %v", instancia.ID, err)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	defer banco.FecharConexaoInstancia(db, tag)

	switch event {
	case "invoice.status_changed":
		handleStatusChanged(w, db, data, instancia)
	default:
		logger.Info(tag, "Instancia %d: evento %s ignorado (iugu_fatura=%s)", instancia.ID, event, iuguID)
		w.WriteHeader(http.StatusOK)
	}
}

func handleStatusChanged(w http.ResponseWriter, db *sql.DB, data map[string]string, instancia dominio.Instancia) {
	id := data["id"]
	status := data["status"]
	externalRef := data["external_reference"]

	if id == "" {
		logger.Aviso(tag, "Instancia %d: data[id] vazio no webhook", instancia.ID)
		w.WriteHeader(http.StatusOK)
		return
	}

	dadosJSON := ""
	if j, ok := data["dados_json"]; ok {
		dadosJSON = j
	} else {
		parts := make([]string, 0, len(data))
		for k, v := range data {
			parts = append(parts, k+"="+v)
		}
		dadosJSON = "{" + strings.Join(parts, ", ") + "}"
	}

	_, err := db.Exec(`INSERT INTO gisp_iugu_gatilhos 
		(id, account_id, external_reference, source, order_id, status, event, dados_json, datetime_received)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		data["account_id"],
		externalRef,
		data["source"],
		data["order_id"],
		status,
		"invoice.status_changed",
		dadosJSON,
		fuso.Agora().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		logger.Aviso(tag, "Instancia %d: erro ao inserir gatilho %s: %v (ja existe?)", instancia.ID, id, err)
	}

	switch status {
	case "paid":
		logger.Info(tag, "Instancia %d: processando paid iugu_fatura=%s ref=%s", instancia.ID, id, truncate(externalRef, 20))
		processarPagamento(w, db, data, id, instancia, "paid")

	case "partially_paid":
		logger.Info(tag, "Instancia %d: processando partially_paid iugu_fatura=%s ref=%s", instancia.ID, id, truncate(externalRef, 20))
		processarPagamento(w, db, data, id, instancia, "partially_paid")

	case "externally_paid":
		logger.Info(tag, "Instancia %d: processando externally_paid iugu_fatura=%s ref=%s", instancia.ID, id, truncate(externalRef, 20))
		processarPagamentoExternal(w, db, data, id, instancia)

	case "canceled":
		logger.Info(tag, "Instancia %d: fatura %s cancelada, ignorando", instancia.ID, id)
		w.WriteHeader(http.StatusOK)

	default:
		logger.Info(tag, "Instancia %d: status %s ignorado para fatura %s", instancia.ID, status, id)
		w.WriteHeader(http.StatusOK)
	}
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func gerarProtocolo(min, max int) int {
	return min + (idCounter() % (max - min + 1))
}

var counter int

func idCounter() int {
	counter++
	return counter
}

func origemPagamento(metodo string) string {
	if cod, ok := codigosOrigem[metodo]; ok {
		return cod
	}
	return "7"
}

func limparNumero(valor string) string {
	r := strings.NewReplacer(".", "", ",", "", "R$", "", " ", "")
	return r.Replace(valor)
}
