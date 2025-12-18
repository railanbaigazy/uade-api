package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/stretchr/testify/assert"
)

func TestNotificationHandler_GetUserNotifications(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	handler := NewNotificationHandler(sqlxDB)

	t.Run("successfully fetch notifications", func(t *testing.T) {
		userID := int64(1)
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "type", "title", "message", "status",
			"agreement_id", "metadata", "created_at", "read_at",
		}).AddRow(
			1, userID, "agreement_accepted", "Agreement Accepted",
			"Your agreement has been accepted", "unread",
			123, nil, now, nil,
		)

		mock.ExpectQuery("SELECT (.+) FROM notifications WHERE user_id = (.+)").
			WithArgs(userID, 50, 0).
			WillReturnRows(rows)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM notifications").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		req := httptest.NewRequest("GET", "/api/notifications", nil)
		req.Header.Set("X-User-ID", "1")
		w := httptest.NewRecorder()

		handler.GetUserNotifications(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Contains(t, response, "notifications")
		assert.Contains(t, response, "unread_count")
	})
}

func TestNotificationHandler_MarkAsRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	handler := NewNotificationHandler(sqlxDB)

	t.Run("successfully mark notification as read", func(t *testing.T) {
		userID := int64(1)
		notificationID := 1
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "type", "title", "message", "status",
			"agreement_id", "metadata", "created_at", "read_at",
		}).AddRow(
			notificationID, userID, "agreement_accepted", "Agreement Accepted",
			"Your agreement has been accepted", "unread",
			123, nil, now, nil,
		)

		mock.ExpectQuery("SELECT \\* FROM notifications WHERE id=(.+)").
			WithArgs(fmt.Sprintf("%d", notificationID)).
			WillReturnRows(rows)

		mock.ExpectExec("UPDATE notifications SET status = (.+), read_at = (.+) WHERE id = (.+)").
			WithArgs(models.NotificationStatusRead, sqlmock.AnyArg(), fmt.Sprintf("%d", notificationID)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := httptest.NewRequest("POST", "/api/notifications/1/read", nil)
		req.Header.Set("X-User-ID", "1")
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		handler.MarkAsRead(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var notification models.Notification
		err = json.NewDecoder(w.Body).Decode(&notification)
		assert.NoError(t, err)
		assert.Equal(t, models.NotificationStatusRead, notification.Status)
	})

	t.Run("unauthorized access", func(t *testing.T) {
		userID := int64(1)
		notificationID := 1
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "type", "title", "message", "status",
			"agreement_id", "metadata", "created_at", "read_at",
		}).AddRow(
			notificationID, 999, "agreement_accepted", "Agreement Accepted",
			"Your agreement has been accepted", "unread",
			123, nil, now, nil,
		)

		mock.ExpectQuery("SELECT \\* FROM notifications WHERE id=(.+)").
			WithArgs(fmt.Sprintf("%d", notificationID)).
			WillReturnRows(rows)

		req := httptest.NewRequest("POST", "/api/notifications/1/read", nil)
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		handler.MarkAsRead(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestNotificationHandler_GetUnreadCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	handler := NewNotificationHandler(sqlxDB)

	t.Run("successfully get unread count", func(t *testing.T) {
		userID := int64(1)
		unreadCount := 5

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM notifications").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(unreadCount))

		req := httptest.NewRequest("GET", "/api/notifications/unread-count", nil)
		req.Header.Set("X-User-ID", "1")
		w := httptest.NewRecorder()

		handler.GetUnreadCount(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, float64(unreadCount), response["unread_count"])
	})
}

func TestNotificationHandler_MarkAllAsRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	handler := NewNotificationHandler(sqlxDB)

	t.Run("successfully mark all as read", func(t *testing.T) {
		userID := int64(1)

		mock.ExpectExec("UPDATE notifications SET status = (.+), read_at = (.+) WHERE user_id = (.+) AND status = 'unread'").
			WithArgs(models.NotificationStatusRead, sqlmock.AnyArg(), userID).
			WillReturnResult(sqlmock.NewResult(0, 3))

		req := httptest.NewRequest("POST", "/api/notifications/read-all", nil)
		req.Header.Set("X-User-ID", "1")
		w := httptest.NewRecorder()

		handler.MarkAllAsRead(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, float64(3), response["updated_count"])
	})
}

func TestNotificationHandler_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	handler := NewNotificationHandler(sqlxDB)

	t.Run("successfully delete notification", func(t *testing.T) {
		userID := int64(1)
		notificationID := 1
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "type", "title", "message", "status",
			"agreement_id", "metadata", "created_at", "read_at",
		}).AddRow(
			notificationID, userID, "agreement_accepted", "Agreement Accepted",
			"Your agreement has been accepted", "read",
			123, nil, now, &now,
		)

		mock.ExpectQuery("SELECT \\* FROM notifications WHERE id=(.+)").
			WithArgs(fmt.Sprintf("%d", notificationID)).
			WillReturnRows(rows)

		mock.ExpectExec("DELETE FROM notifications WHERE id = (.+)").
			WithArgs(fmt.Sprintf("%d", notificationID)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		req := httptest.NewRequest("DELETE", "/api/notifications/1", nil)
		req.Header.Set("X-User-ID", "1")
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		handler.Delete(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
