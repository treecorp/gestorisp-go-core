package helpers

import (
	"encoding/json"
	"net/http"
)

// RespostaJSON representa uma resposta padronizada para endpoints HTTP.
type RespostaJSON struct {
	Sucesso  bool   `json:"sucesso"`
	Mensagem string `json:"mensagem,omitempty"`
	Erro     string `json:"erro,omitempty"`
}

// ResponderJSON codifica v como JSON e escreve na resposta HTTP com o
// status informado, definindo o Content-Type como application/json.
func ResponderJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
