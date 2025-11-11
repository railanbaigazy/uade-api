package handlers

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/railanbaigazy/uade-api/internal/utils"
)

type AuthHandler struct {
	DB  *sqlx.DB
	Cfg *config.Config
}

// helper for JSON error responses
func writeJSONError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func NewAuthHandler(db *sqlx.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{DB: db, Cfg: cfg}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Consolidated validations
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	if input.Name == "" {
		writeJSONError(w, "name is required", http.StatusBadRequest)
		return
	}
	if input.Email == "" {
		writeJSONError(w, "email is required", http.StatusBadRequest)
		return
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		writeJSONError(w, "email is invalid", http.StatusBadRequest)
		return
	}
	if input.Password == "" {
		writeJSONError(w, "password is required", http.StatusBadRequest)
		return
	}
	if len(input.Password) < 6 {
		writeJSONError(w, "password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	_, err = h.DB.Exec(`
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
	`, input.Name, input.Email, hashedPassword)
	if err != nil {
		// handle unique constraint violation for email
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				writeJSONError(w, "email already exists", http.StatusConflict)
				return
			}
		}
		writeJSONError(w, "cannot create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Basic validations for login
	input.Email = strings.TrimSpace(input.Email)
	if input.Email == "" {
		writeJSONError(w, "email is required", http.StatusBadRequest)
		return
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		writeJSONError(w, "email is invalid", http.StatusBadRequest)
		return
	}
	if input.Password == "" {
		writeJSONError(w, "password is required", http.StatusBadRequest)
		return
	}

	var storedHash string
	var userID int
	err := h.DB.QueryRow("SELECT id, password_hash FROM users WHERE email=$1", input.Email).Scan(&userID, &storedHash)
	if err != nil {
		writeJSONError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if !utils.CheckPassword(storedHash, input.Password) {
		writeJSONError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := utils.GenerateJWT(userID, h.Cfg.JWTSecret)
	if err != nil {
		writeJSONError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"token": token}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
