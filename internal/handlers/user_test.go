package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/app/middleware"
	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/stretchr/testify/require"
)

func TestProfileHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	rows := sqlmock.NewRows([]string{"id", "name", "email", "role", "state", "created_at"}).
		AddRow(1, "Test User", "me@example.com", "user", "active", time.Now())

	mock.ExpectQuery("SELECT id, name, email, role, state, created_at FROM users WHERE id=\\$1").
		WithArgs("1").
		WillReturnRows(rows)

	h := NewUserHandler(sqlxDB)

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set("X-User-ID", "1")

	rec := httptest.NewRecorder()
	h.Profile(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var user models.User
	err = json.NewDecoder(rec.Body).Decode(&user)
	require.NoError(t, err)
	require.Equal(t, "me@example.com", user.Email)
	require.Equal(t, int64(1), user.ID)

	err = mock.ExpectationsWereMet()
	require.NoError(t, err, "not all database expectations were met")
}

func TestProfileHandler_WithMiddleware(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	rows := sqlmock.NewRows([]string{"id", "name", "email", "role", "state", "created_at"}).
		AddRow(42, "John Doe", "john@example.com", "user", "active", time.Now())

	mock.ExpectQuery("SELECT id, name, email, role, state, created_at FROM users WHERE id=\\$1").
		WithArgs("42").
		WillReturnRows(rows)

	h := NewUserHandler(sqlxDB)
	secret := "test-secret-key"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 42,
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	})
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	wrappedHandler := middleware.JWTAuth(secret, http.HandlerFunc(h.Profile))

	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "expected 200 status, got %d: %s", rec.Code, rec.Body.String())

	var user models.User
	err = json.NewDecoder(rec.Body).Decode(&user)
	require.NoError(t, err)
	require.Equal(t, "john@example.com", user.Email)
	require.Equal(t, int64(42), user.ID)

	err = mock.ExpectationsWereMet()
	require.NoError(t, err, "not all database expectations were met")
}

func TestProfileHandler_MissingAuth(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	h := NewUserHandler(sqlxDB)
	secret := "test-secret"

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)

	wrappedHandler := middleware.JWTAuth(secret, http.HandlerFunc(h.Profile))

	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Body.String(), "Missing token")
}

func TestProfileHandler_InvalidToken(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	h := NewUserHandler(sqlxDB)
	secret := "test-secret"

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")

	wrappedHandler := middleware.JWTAuth(secret, http.HandlerFunc(h.Profile))

	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Body.String(), "Unauthorized")
}

func TestProfileHandler_UserNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectQuery("SELECT id, name, email, role, state, created_at FROM users WHERE id=\\$1").
		WithArgs("999").
		WillReturnError(sql.ErrNoRows)

	h := NewUserHandler(sqlxDB)
	secret := "test-secret"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 999,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(secret))

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	wrappedHandler := middleware.JWTAuth(secret, http.HandlerFunc(h.Profile))

	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "User not found")
}
