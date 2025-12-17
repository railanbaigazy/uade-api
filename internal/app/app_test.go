package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/railanbaigazy/uade-api/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestSetupRoutes(t *testing.T) {
	// sqlmock DB
	rawDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = rawDB.Close() })

	db := sqlx.NewDb(rawDB, "sqlmock")

	cfg := &config.Config{
		JWTSecret: "test-secret",
		Env:       "test",
	}

	a := &App{
		DB:  db,
		Cfg: cfg,
		// Publisher: nil, // если у тебя появится поле Publisher в App — оставь nil
	}

	mux := a.SetupRoutes()

	type tc struct {
		name      string
		method    string
		path      string
		body      any
		setupMock func()
		wantCode  int
	}

	tests := []tc{
		{
			name:     "healthz",
			method:   http.MethodGet,
			path:     "/healthz",
			body:     nil,
			wantCode: http.StatusOK,
		},
		{
			name:   "register_endpoint",
			method: http.MethodPost,
			path:   "/api/auth/register",
			body: map[string]any{
				"name":     "Lender",
				"email":    "lender@test.com",
				"password": "secret123",
			},
			setupMock: func() {
				// Register делает Exec(INSERT INTO users ...)
				mock.ExpectExec(`(?is)insert\s+into\s+users`).
					WithArgs("Lender", "lender@test.com", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantCode: http.StatusCreated,
		},
		{
			name:   "login_endpoint",
			method: http.MethodPost,
			path:   "/api/auth/login",
			body: map[string]any{
				"email":    "lender@test.com",
				"password": "secret123",
			},
			setupMock: func() {
				hash, err := utils.HashPassword("secret123")
				require.NoError(t, err)

				mock.ExpectQuery(`(?is)select\s+id,\s*password_hash\s+from\s+users\s+where\s+email=`).
					WithArgs("lender@test.com").
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "password_hash"}).
							AddRow(1, hash),
					)
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			var bodyBytes []byte
			if tt.body != nil {
				bodyBytes, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(bodyBytes))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code, "[%s] %s -> %d body=%s", tt.method, tt.path, rec.Code, rec.Body.String())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
