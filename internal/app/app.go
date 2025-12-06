package app

import (
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/app/middleware"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/railanbaigazy/uade-api/internal/handlers"
	"github.com/railanbaigazy/uade-api/internal/mq"
)

type App struct {
	DB        *sqlx.DB
	Cfg       *config.Config
	Publisher mq.Publisher
}

func New(db *sqlx.DB, cfg *config.Config) *App {
	pub, err := mq.NewPublisher(cfg.RabbitURL, cfg.RabbitExchange)
	if err != nil {
		log.Printf("failed to init RabbitMQ publisher: %v", err)
	}

	return &App{
		DB:        db,
		Cfg:       cfg,
		Publisher: pub,
	}
}

func (a *App) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handlers.HealthzHandler)

	authHandler := handlers.NewAuthHandler(a.DB, a.Cfg)
	userHandler := handlers.NewUserHandler(a.DB)
	postHandler := handlers.NewPostHandler(a.DB)
	agreementHandler := handlers.NewAgreementHandler(a.DB, a.Publisher)

	// Auth
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	// Users
	mux.Handle("GET /api/users/me",
		middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(userHandler.Profile)),
	)

	// Posts
	mux.Handle("GET /api/posts",
		middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.GetAll)))
	mux.Handle("POST /api/posts",
		middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.Create)))
	mux.Handle("PUT /api/posts/{id}",
		middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.Update)))
	mux.Handle("DELETE /api/posts/{id}",
		middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.Delete)))

	// Agreements
	mux.Handle("GET /api/agreements",
		middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.GetUserAgreements)))
	mux.Handle("GET /api/agreements/{id}",
		middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.GetByID)))
	mux.Handle("POST /api/agreements",
		middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.Create)))
	mux.Handle("POST /api/agreements/{id}/accept",
		middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.Accept)))
	mux.Handle("POST /api/agreements/{id}/cancel",
		middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.Cancel)))
	mux.Handle("PUT /api/agreements/{id}/contract",
		middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.UpdateContract)))

	return mux
}
