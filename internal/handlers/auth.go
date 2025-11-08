package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/railanbaigazy/uade-api/internal/utils"
)

type AuthHandler struct {
	DB  *sqlx.DB
	Cfg *config.Config
}

func NewAuthHandler(db *sqlx.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{DB: db, Cfg: cfg}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// password hashing
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	_, err = h.DB.Exec(`
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
	`, req.Name, req.Email, hash)

	if err != nil {
		http.Error(w, "Cannot create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(`{"status": "ok"}`)); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	var user struct {
		ID           int    `db:"id"`
		Name         string `db:"name"`
		Email        string `db:"email"`
		PasswordHash string `db:"password_hash"`
		Role         string `db:"role"`
	}

	err := h.DB.Get(&user, "SELECT * FROM users WHERE email=$1", req.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !utils.CheckPassword(user.PasswordHash, req.Password) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// generate JWT token
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(), // срок действия 24 часа
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.Cfg.JWTSecret))
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	}); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
