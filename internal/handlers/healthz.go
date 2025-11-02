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

	json.NewEncoder(w).Encode(status)
}
