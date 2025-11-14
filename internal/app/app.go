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
	postHandler := handlers.NewPostHandler(a.DB)

	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	mux.Handle("GET /api/users/me", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(userHandler.Profile)))

	// POSTS CRUD
	mux.Handle("GET /api/posts", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.GetAll)))
	mux.Handle("POST /api/posts", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.Create)))
	mux.Handle("PUT /api/posts/{id}", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.Update)))
	mux.Handle("DELETE /api/posts/{id}", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.Delete)))

	return mux
}
