package app

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/railanbaigazy/uade-api/internal/app/middleware"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/railanbaigazy/uade-api/internal/handlers"
	"github.com/railanbaigazy/uade-api/internal/rabbitmq"
)

type App struct {
	DB        *sqlx.DB
	Cfg       *config.Config
	Publisher *rabbitmq.Publisher
}

func New(db *sqlx.DB, cfg *config.Config, pub *rabbitmq.Publisher) *App {
	return &App{DB: db, Cfg: cfg, Publisher: pub}
}

func (a *App) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handlers.HealthzHandler)
	mux.Handle("GET /metrics", promhttp.Handler())

	authHandler := handlers.NewAuthHandler(a.DB, a.Cfg)
	userHandler := handlers.NewUserHandler(a.DB)
	postHandler := handlers.NewPostHandler(a.DB)
	notificationHandler := handlers.NewNotificationHandler(a.DB)

	agreementHandler := handlers.NewAgreementHandler(a.DB, a.Publisher, a.Publisher)

	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	mux.Handle("GET /api/users/me", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(userHandler.Profile)))

	mux.Handle("GET /api/posts", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.GetAll)))
	mux.Handle("POST /api/posts", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.Create)))
	mux.Handle("PUT /api/posts/{id}", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.Update)))
	mux.Handle("DELETE /api/posts/{id}", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(postHandler.Delete)))

	mux.Handle("GET /api/agreements", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.GetUserAgreements)))
	mux.Handle("GET /api/agreements/{id}", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.GetByID)))
	mux.Handle("POST /api/agreements", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.Create)))
	mux.Handle("POST /api/agreements/{id}/accept", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.Accept)))
	mux.Handle("POST /api/agreements/{id}/cancel", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.Cancel)))
	mux.Handle("POST /api/agreements/{id}/contract/generate", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.GenerateContract)))
	mux.Handle("PUT /api/agreements/{id}/contract", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(agreementHandler.UpdateContract)))

	mux.Handle("GET /api/notifications", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(notificationHandler.GetUserNotifications)))
	mux.Handle("GET /api/notifications/unread-count", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(notificationHandler.GetUnreadCount)))
	mux.Handle("GET /api/notifications/{id}", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(notificationHandler.GetByID)))
	mux.Handle("POST /api/notifications/{id}/read", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(notificationHandler.MarkAsRead)))
	mux.Handle("POST /api/notifications/read-all", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(notificationHandler.MarkAllAsRead)))
	mux.Handle("DELETE /api/notifications/{id}", middleware.JWTAuth(a.Cfg.JWTSecret, http.HandlerFunc(notificationHandler.Delete)))

	return mux
}
