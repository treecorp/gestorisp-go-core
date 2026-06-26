package api

import (
	"encoding/json"
	"net/http"
)

type resposta struct {
	Sucesso  bool   `json:"sucesso"`
	Mensagem string `json:"mensagem,omitempty"`
	Erro     string `json:"erro,omitempty"`
}

func responderJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
