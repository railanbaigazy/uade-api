package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/railanbaigazy/uade-api/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestRegisterAndLogin_SQLMock(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	db := sqlx.NewDb(sqlDB, "postgres")
	defer db.Close()

	cfg := &config.Config{JWTSecret: "test-secret"}
	h := NewAuthHandler(db, cfg)

	mock.ExpectExec("INSERT INTO users").WithArgs("TestUser", "user@example.com", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))

	registerBody := `{"name":"TestUser","email":"user@example.com","password":"12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(registerBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	password := "12345678"
	hashed, err := utils.HashPassword(password)
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(1, hashed)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, password_hash FROM users WHERE email=$1")).WithArgs("user@example.com").WillReturnRows(rows)

	loginBody := `{"email":"user@example.com","password":"12345678"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()

	h.Login(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var resp map[string]string
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	tokenStr, ok := resp["token"]
	require.True(t, ok)
	require.NotEmpty(t, tokenStr)

	parsed, err := jwt.ParseWithClaims(tokenStr, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	require.Equal(t, float64(1), claims["user_id"])

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestRegisterValidation(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	db := sqlx.NewDb(sqlDB, "postgres")
	defer db.Close()

	cfg := &config.Config{JWTSecret: "test-secret"}
	h := NewAuthHandler(db, cfg)

	body := `{"name":"","email":"user@example.com","password":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "name is required")

	body = `{"name":"Test","email":"invalid","password":"123456"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.Register(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "email is invalid")

	body = `{"name":"Test","email":"t@example.com","password":"123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.Register(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "password must be at least 6 characters")

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestLoginValidation(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	db := sqlx.NewDb(sqlDB, "postgres")
	defer db.Close()

	cfg := &config.Config{JWTSecret: "test-secret"}
	h := NewAuthHandler(db, cfg)

	body := `{"email":"","password":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "email is required")

	body = `{"email":"not-email","password":"123456"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.Login(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "email is invalid")

	body = `{"email":"user@example.com","password":""}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.Login(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "password is required")

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}
