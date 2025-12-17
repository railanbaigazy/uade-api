package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/railanbaigazy/uade-api/internal/app"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/railanbaigazy/uade-api/internal/rabbitmq"
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

	mq, err := rabbitmq.Connect(cfg.AMQPURL)
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer mq.Close()

	if err := rabbitmq.DeclareTopology(mq.Ch); err != nil {
		log.Fatal("Failed to declare RabbitMQ topology:", err)
	}

	pub := rabbitmq.NewPublisher(mq.Ch)

	a := app.New(db, cfg, pub)
	mux := a.SetupRoutes()

	fmt.Println("Uade API running on port:", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
