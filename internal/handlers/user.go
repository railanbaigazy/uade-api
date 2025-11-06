package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/app/models"
)

type UserHandler struct {
	DB *sqlx.DB
}

func NewUserHandler(db *sqlx.DB) *UserHandler {
	return &UserHandler{DB: db}
}

func (h *UserHandler) Profile(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	var user models.User
	err := h.DB.Get(&user,
		"SELECT id, name, email, role, state, created_at FROM users WHERE id=$1", userID)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(user)
}
