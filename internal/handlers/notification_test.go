package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/railanbaigazy/uade-api/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestNotificationHandler_GetAll_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewNotificationHandler(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "message", "read", "created_at", "read_at", "metadata"}).
		AddRow(1, 10, "agreement_accepted", "Agreement Accepted", "Your agreement has been accepted", false, time.Now(), nil, `{"agreement_id": 1}`).
		AddRow(2, 10, "payment_reminder", "Payment Reminder", "Payment due soon", true, time.Now(), time.Now(), `{"agreement_id": 2}`)

	mock.ExpectQuery(`SELECT id, user_id, type, title, message, read, created_at, read_at, metadata`).
		WithArgs(int64(10)).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	req.Header.Set("X-User-ID", "10")
	rec := httptest.NewRecorder()

	h.GetAll(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var notifications []models.Notification
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&notifications))
	require.Len(t, notifications, 2)
	require.Equal(t, "Agreement Accepted", notifications[0].Title)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationHandler_GetAll_WithReadFilter(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewNotificationHandler(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "message", "read", "created_at", "read_at", "metadata"}).
		AddRow(2, 10, "payment_reminder", "Payment Reminder", "Payment due soon", true, time.Now(), time.Now(), `{"agreement_id": 2}`)

	mock.ExpectQuery(`SELECT id, user_id, type, title, message, read, created_at, read_at, metadata`).
		WithArgs(int64(10)).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications?read=true", nil)
	req.Header.Set("X-User-ID", "10")
	rec := httptest.NewRecorder()

	h.GetAll(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var notifications []models.Notification
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&notifications))
	require.Len(t, notifications, 1)
	require.True(t, notifications[0].Read)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationHandler_GetAll_InvalidUserID(t *testing.T) {
	db, _ := utils.NewSQLXMock(t)
	h := NewNotificationHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	req.Header.Set("X-User-ID", "invalid")
	rec := httptest.NewRecorder()

	h.GetAll(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "invalid user id")
}

func TestNotificationHandler_GetByID_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewNotificationHandler(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "message", "read", "created_at", "read_at", "metadata"}).
		AddRow(1, 10, "agreement_accepted", "Agreement Accepted", "Your agreement has been accepted", false, time.Now(), nil, `{"agreement_id": 1}`)

	mock.ExpectQuery(`SELECT id, user_id, type, title, message, read, created_at, read_at, metadata`).
		WithArgs("1", int64(10)).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/1", nil)
	req.Header.Set("X-User-ID", "10")
	rec := httptest.NewRecorder()

	h.GetByID(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var notification models.Notification
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&notification))
	require.Equal(t, int64(1), notification.ID)
	require.Equal(t, "Agreement Accepted", notification.Title)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationHandler_GetByID_NotFound(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewNotificationHandler(db)

	mock.ExpectQuery(`SELECT id, user_id, type, title, message, read, created_at, read_at, metadata`).
		WithArgs("1", int64(10)).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/1", nil)
	req.Header.Set("X-User-ID", "10")
	rec := httptest.NewRecorder()

	h.GetByID(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "notification not found")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationHandler_MarkAsRead_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewNotificationHandler(db)

	// First query to get notification
	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "message", "read", "created_at", "read_at", "metadata"}).
		AddRow(1, 10, "agreement_accepted", "Agreement Accepted", "Your agreement has been accepted", false, time.Now(), nil, `{"agreement_id": 1}`)

	mock.ExpectQuery(`SELECT id, user_id, type, title, message, read, created_at, read_at, metadata`).
		WithArgs("1", int64(10)).
		WillReturnRows(rows)

	// Update query
	mock.ExpectExec(`UPDATE notifications`).
		WithArgs("1", int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPatch, "/api/notifications/1/read", nil)
	req.Header.Set("X-User-ID", "10")
	rec := httptest.NewRecorder()

	h.MarkAsRead(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var notification models.Notification
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&notification))
	require.True(t, notification.Read)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationHandler_MarkAsRead_AlreadyRead(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewNotificationHandler(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "message", "read", "created_at", "read_at", "metadata"}).
		AddRow(1, 10, "agreement_accepted", "Agreement Accepted", "Your agreement has been accepted", true, time.Now(), time.Now(), `{"agreement_id": 1}`)

	mock.ExpectQuery(`SELECT id, user_id, type, title, message, read, created_at, read_at, metadata`).
		WithArgs("1", int64(10)).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodPatch, "/api/notifications/1/read", nil)
	req.Header.Set("X-User-ID", "10")
	rec := httptest.NewRecorder()

	h.MarkAsRead(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "already marked as read")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationHandler_MarkAllAsRead_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewNotificationHandler(db)

	mock.ExpectExec(`UPDATE notifications`).
		WithArgs(int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 3))

	req := httptest.NewRequest(http.MethodPatch, "/api/notifications/read-all", nil)
	req.Header.Set("X-User-ID", "10")
	rec := httptest.NewRecorder()

	h.MarkAllAsRead(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)

	require.NoError(t, mock.ExpectationsWereMet())
}
