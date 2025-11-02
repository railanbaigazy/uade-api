package app

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/handlers"
)

type App struct {
	DB *sqlx.DB
}

func New(db *sqlx.DB) *App {
	return &App{DB: db}
}

func (a *App) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handlers.HealthzHandler)

	return mux
}
