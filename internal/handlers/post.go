package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/app/models"
)

type PostHandler struct {
	DB *sqlx.DB
}

func NewPostHandler(db *sqlx.DB) *PostHandler {
	return &PostHandler{DB: db}
}

// GET /api/posts
func (h *PostHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id, title, content, author_id, created_at FROM posts ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []models.Post

	for rows.Next() {
		var p models.Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.AuthorID, &p.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		posts = append(posts, p)
	}

	json.NewEncoder(w).Encode(posts)
}

// POST /api/posts
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	var p models.Post
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	query := `INSERT INTO posts (title, content, author_id, created_at)
              VALUES ($1, $2, $3, NOW()) RETURNING id, created_at`

	err := h.DB.QueryRow(query, p.Title, p.Content, userID).
		Scan(&p.ID, &p.CreatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.AuthorID = userID

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

// PUT /api/posts/{id}
func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var p models.Post
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var authorID int64
	err := h.DB.QueryRow("SELECT author_id FROM posts WHERE id = $1", id).
		Scan(&authorID)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	if userID != authorID {
		http.Error(w, "not allowed", http.StatusForbidden)
		return
	}

	_, err = h.DB.Exec("UPDATE posts SET title=$1, content=$2 WHERE id=$3",
		p.Title, p.Content, id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// DELETE /api/posts/{id}
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var authorID int64
	err := h.DB.QueryRow("SELECT author_id FROM posts WHERE id = $1", id).
		Scan(&authorID)

	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	if userID != authorID {
		http.Error(w, "not allowed", http.StatusForbidden)
		return
	}

	_, err = h.DB.Exec("DELETE FROM posts WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
