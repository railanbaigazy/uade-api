package handlers

import (
	"encoding/json"
	"net/http"
)

func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := map[string]string{
		"app": "ok",
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
