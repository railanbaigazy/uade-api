package app

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/app/middleware"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/railanbaigazy/uade-api/internal/handlers"
)

type App struct {
	DB  *sqlx.DB
	Cfg *config.Config
}

func New(db *sqlx.DB, cfg *config.Config) *App {
	return &App{DB: db, Cfg: cfg}
}

func (a *App) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handlers.HealthzHandler)

	authHandler := handlers.NewAuthHandler(a.DB, a.Cfg)
	userHandler := handlers.NewUserHandler(a.DB)

	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	// JWT middleware
	mux.Handle("/api/users/me", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(userHandler.Profile)))

	return mux
}
