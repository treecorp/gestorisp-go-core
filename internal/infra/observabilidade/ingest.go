package observabilidade

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gestor/internal/infra/logger"
)

var metricas Metricas

func HandlerIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo nao permitido", http.StatusMethodNotAllowed)
		return
	}

	var entry LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		return
	}

	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format("2006/01/02 15:04:05")
	}

	hub.Broadcast(entry)
	metricas.Ingerir(entry)

	w.WriteHeader(http.StatusOK)
}

func HandlerMetricas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	snap := metricas.Snapshot()
	json.NewEncoder(w).Encode(snap)
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

func EnviarLog(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	url := ingestorURL
	if url == "" {
		return
	}

	go func() {
		resp, err := httpClient.Post(url+"/api/ingest", "application/json", bytes.NewReader(data))
		if err != nil {
			logger.Aviso("observabilidade", "Falha ao enviar log para dashboard: %v", err)
			return
		}
		resp.Body.Close()
	}()
}

var ingestorURL string

func ConfigurarIngestor(url string) {
	ingestorURL = url
	if url != "" {
		logger.Info("observabilidade", "Ingestor configurado: %s", url)
	}
}

type LogIngestor struct{}

func (LogIngestor) WriteLog(nivel, tag, msg string) {
	EnviarLog(LogEntry{
		Timestamp: time.Now().Format("2006/01/02 15:04:05"),
		Nivel:     nivel,
		Tag:       tag,
		Mensagem:  msg,
		Servico:   obterServico(),
	})
}

var nomeServico string

func DefinirServico(nome string) {
	nomeServico = nome
}

func obterServico() string {
	if nomeServico != "" {
		return nomeServico
	}
	return "desconhecido"
}

func LogCompleto(nivel, tag, msg string, instanciaID int, duracaoMs int64, args ...interface{}) {
	texto := fmt.Sprintf(msg, args...)
	EnviarLog(LogEntry{
		Timestamp: time.Now().Format("2006/01/02 15:04:05"),
		Nivel:     nivel,
		Tag:       tag,
		Instancia: instanciaID,
		Mensagem:  texto,
		DuracaoMs: duracaoMs,
		Servico:   obterServico(),
	})
}
