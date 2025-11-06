package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/railanbaigazy/uade-api/internal/config"
)

type AuthHandler struct {
	DB  *sqlx.DB
	Cfg *config.Config
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
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	query := `INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3)
			  RETURNING id, role, state, created_at`
	var user models.User
	err := h.DB.QueryRowx(query, input.Name, input.Email, string(hashed)).
		Scan(&user.ID, &user.Role, &user.State, &user.CreatedAt)
	if err != nil {
		http.Error(w, "Cannot create user", http.StatusInternalServerError)
		return
	}
	user.Name = input.Name
	user.Email = input.Email

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var user models.User
	err := h.DB.Get(&user, "SELECT * FROM users WHERE email=$1", input.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(h.Cfg.JWTSecret))

	json.NewEncoder(w).Encode(map[string]string{"token": tokenStr})
}
