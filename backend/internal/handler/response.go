package handler

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

func respondWithJSON(w http.ResponseWriter, status int, paylaod interface{}) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(paylaod)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write(data)
	// _ = json.NewEncoder(w).Encode(Response{Success: true, Data: data})
}

func respondWithError(w http.ResponseWriter, status int, message string) {
	respondWithJSON(w, status, errorResponse{
		Error:   message,
		Details: nil, // fix after error refactor
	})
}
