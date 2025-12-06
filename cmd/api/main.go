package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/railanbaigazy/uade-api/internal/app"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/railanbaigazy/uade-api/internal/metrics"
)

func main() {
	cfg := config.Load()
	metrics.Register()

	db, err := sqlx.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Database not reachable:", err)
	}

	a := app.New(db, cfg)
	apiMux := a.SetupRoutes()

	rootMux := http.NewServeMux()
	rootMux.Handle("/metrics", promhttp.Handler())
	rootMux.Handle("/", apiMux)

	fmt.Println("Uade API running on port:", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, rootMux))

}
