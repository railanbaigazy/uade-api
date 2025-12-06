package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/railanbaigazy/uade-api/internal/utils"
)

type NotificationHandler struct {
	DB *sqlx.DB
}

func NewNotificationHandler(db *sqlx.DB) *NotificationHandler {
	return &NotificationHandler{
		DB: db,
	}
}

// GetAll retrieves all notifications for the authenticated user
func (h *NotificationHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		utils.WriteJSONError(w, "invalid user id", http.StatusBadRequest)
		return
	}

	readFilter := r.URL.Query().Get("read")
	query := `
		SELECT id, user_id, type, title, message, read, created_at, read_at, metadata
		FROM notifications
		WHERE user_id = $1
	`

	args := []any{userID}
	switch readFilter {
	case "true":
		query += " AND read = true"
	case "false":
		query += " AND read = false"
	}

	query += " ORDER BY created_at DESC"

	notifications := make([]models.Notification, 0)
	if err := h.DB.Select(&notifications, query, args...); err != nil {
		utils.WriteJSONError(w, "failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(notifications); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// GetByID retrieves a specific notification by ID
func (h *NotificationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		utils.WriteJSONError(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var notification models.Notification
	query := `
		SELECT id, user_id, type, title, message, read, created_at, read_at, metadata
		FROM notifications
		WHERE id = $1 AND user_id = $2
	`

	err = h.DB.Get(&notification, query, id, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONError(w, "notification not found", http.StatusNotFound)
			return
		}
		utils.WriteJSONError(w, "failed to fetch notification", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(notification); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// MarkAsRead marks a notification as read
func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		utils.WriteJSONError(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var notification models.Notification
	query := `
		SELECT id, user_id, type, title, message, read, created_at, read_at, metadata
		FROM notifications
		WHERE id = $1 AND user_id = $2
	`

	err = h.DB.Get(&notification, query, id, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONError(w, "notification not found", http.StatusNotFound)
			return
		}
		utils.WriteJSONError(w, "failed to fetch notification", http.StatusInternalServerError)
		return
	}

	if notification.Read {
		utils.WriteJSONError(w, "notification already marked as read", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec(`
		UPDATE notifications 
		SET read = true, read_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		utils.WriteJSONError(w, "failed to mark notification as read", http.StatusInternalServerError)
		return
	}

	notification.Read = true

	if err := json.NewEncoder(w).Encode(notification); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// MarkAllAsRead marks all notifications for the user as read
func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		utils.WriteJSONError(w, "invalid user id", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec(`
		UPDATE notifications 
		SET read = true, read_at = NOW()
		WHERE user_id = $1 AND read = false
	`, userID)
	if err != nil {
		utils.WriteJSONError(w, "failed to mark notifications as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
