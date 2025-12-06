package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/railanbaigazy/uade-api/internal/app"
	"github.com/railanbaigazy/uade-api/internal/config"
<<<<<<< HEAD
	"github.com/railanbaigazy/uade-api/internal/mq"
=======
	"github.com/railanbaigazy/uade-api/internal/metrics"
>>>>>>> 0a5a1fb5cd5f313529f0a404b4d56f8823e42c8a
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

	// Start RabbitMQ consumer
	consumer, err := mq.NewConsumer(cfg.RabbitURL, cfg.RabbitExchange, db)
	if err != nil {
		log.Printf("Failed to create RabbitMQ consumer: %v (continuing without consumer)", err)
	} else {
		defer consumer.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start consumer in a goroutine
		go func() {
			if err := consumer.Start(ctx); err != nil {
				log.Printf("Consumer error: %v", err)
			}
		}()

		// Handle graceful shutdown
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan
			log.Println("Shutting down consumer...")
			cancel()
		}()
	}

	a := app.New(db, cfg)
	apiMux := a.SetupRoutes()

	rootMux := http.NewServeMux()
	rootMux.Handle("/metrics", promhttp.Handler())
	rootMux.Handle("/", apiMux)

	fmt.Println("Uade API running on port:", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, rootMux))

}
