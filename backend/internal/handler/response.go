package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type errorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

func respondWithJSON(w http.ResponseWriter, status int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal response", "error", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"something went wrong"}`))
		return
	}

	w.Header().Set("Contnet-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func respondWithError(w http.ResponseWriter, status int, message string) {
	respondWithJSON(w, status, errorResponse{
		Error:   message,
		Details: nil,
	})
}
