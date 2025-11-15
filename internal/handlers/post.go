package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/railanbaigazy/uade-api/internal/utils"
)

type PostHandler struct {
	DB *sqlx.DB
}

func NewPostHandler(db *sqlx.DB) *PostHandler {
	return &PostHandler{DB: db}
}

func (h *PostHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	posts := make([]models.Post, 0)

	query := `SELECT id, title, content, type, author_id, created_at 
	          FROM posts 
	          ORDER BY created_at DESC`

	if err := h.DB.Select(&posts, query); err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(posts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	var p models.Post
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	query := `
		INSERT INTO posts (title, content, type, author_id, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, created_at
	`

	if err := h.DB.Get(&p, query, p.Title, p.Content, p.Type, userID); err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.AuthorID = userID

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var p models.Post
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var authorID int64
	if err := h.DB.Get(&authorID, "SELECT author_id FROM posts WHERE id=$1", id); err != nil {
		utils.WriteJSONError(w, "not found", http.StatusNotFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	if userID != authorID {
		utils.WriteJSONError(w, "not allowed", http.StatusForbidden)
		return
	}

	_, err := h.DB.Exec(
		"UPDATE posts SET title=$1, content=$2 WHERE id=$3",
		p.Title, p.Content, id,
	)
	if err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var authorID int64
	if err := h.DB.Get(&authorID, "SELECT author_id FROM posts WHERE id=$1", id); err != nil {
		utils.WriteJSONError(w, "not found", http.StatusNotFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	if userID != authorID {
		utils.WriteJSONError(w, "not allowed", http.StatusForbidden)
		return
	}

	_, err := h.DB.Exec("DELETE FROM posts WHERE id=$1", id)
	if err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
