package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSetupRoutes(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}
	db := &sqlx.DB{} // пустой объект без подключения

	a := New(db, cfg)
	mux := a.SetupRoutes()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"healthz works", http.MethodGet, "/healthz", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)
			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
