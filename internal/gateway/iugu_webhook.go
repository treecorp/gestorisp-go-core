package gateway

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
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

	event := r.PostFormValue("event")
	dataJSON := r.PostFormValue("data")

	if event == "" {
		http.Error(w, "event nao informado", http.StatusBadRequest)
		return
	}
	if dataJSON == "" {
		http.Error(w, "data nao informado", http.StatusBadRequest)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		logger.Aviso(tag, "Instancia %d: erro ao decodificar data JSON: %v", instancia.ID, err)
		http.Error(w, "data JSON invalido", http.StatusBadRequest)
		return
	}

	logger.Info(tag, "Instancia %d: webhook recebido event=%s", instancia.ID, event)

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
		logger.Info(tag, "Instancia %d: evento %s ignorado", instancia.ID, event)
		w.WriteHeader(http.StatusOK)
	}
}

func handleStatusChanged(w http.ResponseWriter, db *sql.DB, data map[string]interface{}, instancia dominio.Instancia) {
	id, _ := data["id"].(string)
	status, _ := data["status"].(string)
	externalRef, _ := data["external_reference"].(string)

	if id == "" {
		logger.Aviso(tag, "Instancia %d: id nao informado no data", instancia.ID)
		w.WriteHeader(http.StatusOK)
		return
	}

	_, err := db.Exec(`INSERT INTO gisp_iugu_gatilhos 
		(id, account_id, external_reference, source, order_id, status, event, dados_json, datetime_received)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		valorString(data, "account_id"),
		externalRef,
		valorString(data, "source"),
		valorString(data, "order_id"),
		status,
		"invoice.status_changed",
		valorString(data, "dados_json"),
		fuso.Agora().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		logger.Erro(tag, "Instancia %d: erro ao inserir gatilho %s: %v", instancia.ID, id, err)
	}

	switch status {
	case "paid":
		logger.Info(tag, "Instancia %d: processando pagamento paid, fatura %s", instancia.ID, id)
		processarPagamento(w, db, data, id, instancia, "paid")

	case "partially_paid":
		logger.Info(tag, "Instancia %d: processando pagamento partially_paid, fatura %s", instancia.ID, id)
		processarPagamento(w, db, data, id, instancia, "partially_paid")

	case "externally_paid":
		logger.Info(tag, "Instancia %d: processando pagamento externally_paid, fatura %s", instancia.ID, id)
		processarPagamentoExternal(w, db, data, id, instancia)

	case "canceled":
		logger.Info(tag, "Instancia %d: fatura %s cancelada, ignorando", instancia.ID, id)
		w.WriteHeader(http.StatusOK)

	default:
		logger.Info(tag, "Instancia %d: status %s ignorado para fatura %s", instancia.ID, status, id)
		w.WriteHeader(http.StatusOK)
	}
}

func valorString(data map[string]interface{}, chave string) string {
	if v, ok := data[chave]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func valorInt(data map[string]interface{}, chave string) int {
	if v, ok := data[chave]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return int(n)
		case string:
			i, _ := strconv.Atoi(n)
			return i
		}
	}
	return 0
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
