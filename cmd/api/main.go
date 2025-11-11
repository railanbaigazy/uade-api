package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/railanbaigazy/uade-api/internal/app"
	"github.com/railanbaigazy/uade-api/internal/config"
)

func main() {
	cfg := config.Load()

	db, err := sqlx.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Database not reachable:", err)
	}

	a := app.New(db, cfg)
	mux := a.SetupRoutes()

	fmt.Println("Uade API running on port:", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
