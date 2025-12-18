package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/railanbaigazy/uade-api/internal/utils"
)

type NotificationHandler struct {
	DB *sqlx.DB
}

func NewNotificationHandler(db *sqlx.DB) *NotificationHandler {
	return &NotificationHandler{DB: db}
}

// GetUserNotifications returns all notifications for the authenticated user
func (h *NotificationHandler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	statusFilter := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	query := `
		SELECT 
			id, user_id, type, title, message, status,
			agreement_id, metadata, created_at, read_at
		FROM notifications
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argCount := 1

	if statusFilter != "" {
		argCount++
		query += " AND status = $" + strconv.Itoa(argCount)
		args = append(args, statusFilter)
	}

	query += " ORDER BY created_at DESC"

	argCount++
	query += " LIMIT $" + strconv.Itoa(argCount)
	args = append(args, limit)

	argCount++
	query += " OFFSET $" + strconv.Itoa(argCount)
	args = append(args, offset)

	notifications := make([]models.Notification, 0)
	if err := h.DB.Select(&notifications, query, args...); err != nil {
		utils.WriteJSONError(w, "failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	// Get unread count
	var unreadCount int
	_ = h.DB.Get(&unreadCount, `
		SELECT COUNT(*) FROM notifications 
		WHERE user_id = $1 AND status = 'unread'
	`, userID)

	response := map[string]interface{}{
		"notifications": notifications,
		"unread_count":  unreadCount,
		"limit":         limit,
		"offset":        offset,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// GetByID returns a specific notification
func (h *NotificationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	var notification models.Notification
	query := `
		SELECT 
			id, user_id, type, title, message, status,
			agreement_id, metadata, created_at, read_at
		FROM notifications
		WHERE id = $1
	`

	err := h.DB.Get(&notification, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONError(w, "notification not found", http.StatusNotFound)
			return
		}
		utils.WriteJSONError(w, "failed to fetch notification", http.StatusInternalServerError)
		return
	}

	// Check authorization
	if notification.UserID != userID {
		utils.WriteJSONError(w, "not authorized to view this notification", http.StatusForbidden)
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
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	var notification models.Notification
	err := h.DB.Get(&notification, "SELECT * FROM notifications WHERE id=$1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONError(w, "notification not found", http.StatusNotFound)
			return
		}
		utils.WriteJSONError(w, "failed to fetch notification", http.StatusInternalServerError)
		return
	}

	// Check authorization
	if notification.UserID != userID {
		utils.WriteJSONError(w, "not authorized to modify this notification", http.StatusForbidden)
		return
	}

	if notification.Status == models.NotificationStatusRead {
		utils.WriteJSONError(w, "notification already marked as read", http.StatusBadRequest)
		return
	}

	now := time.Now()
	_, err = h.DB.Exec(`
		UPDATE notifications 
		SET status = $1, read_at = $2
		WHERE id = $3
	`, models.NotificationStatusRead, now, id)
	if err != nil {
		utils.WriteJSONError(w, "failed to mark notification as read", http.StatusInternalServerError)
		return
	}

	notification.Status = models.NotificationStatusRead
	notification.ReadAt = &now

	if err := json.NewEncoder(w).Encode(notification); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// MarkAllAsRead marks all unread notifications as read for the user
func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	now := time.Now()
	result, err := h.DB.Exec(`
		UPDATE notifications 
		SET status = $1, read_at = $2
		WHERE user_id = $3 AND status = 'unread'
	`, models.NotificationStatusRead, now, userID)
	if err != nil {
		utils.WriteJSONError(w, "failed to mark notifications as read", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()

	response := map[string]interface{}{
		"message":       "notifications marked as read",
		"updated_count": rowsAffected,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// Delete deletes a notification
func (h *NotificationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	var notification models.Notification
	err := h.DB.Get(&notification, "SELECT * FROM notifications WHERE id=$1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONError(w, "notification not found", http.StatusNotFound)
			return
		}
		utils.WriteJSONError(w, "failed to fetch notification", http.StatusInternalServerError)
		return
	}

	// Check authorization
	if notification.UserID != userID {
		utils.WriteJSONError(w, "not authorized to delete this notification", http.StatusForbidden)
		return
	}

	_, err = h.DB.Exec("DELETE FROM notifications WHERE id = $1", id)
	if err != nil {
		utils.WriteJSONError(w, "failed to delete notification", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetUnreadCount returns the count of unread notifications
func (h *NotificationHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	var count int
	err := h.DB.Get(&count, `
		SELECT COUNT(*) FROM notifications 
		WHERE user_id = $1 AND status = 'unread'
	`, userID)
	if err != nil {
		utils.WriteJSONError(w, "failed to get unread count", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"unread_count": count,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}
