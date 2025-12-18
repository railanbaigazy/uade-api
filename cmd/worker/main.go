package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/railanbaigazy/uade-api/internal/contracts"
	"github.com/railanbaigazy/uade-api/internal/rabbitmq"
	"github.com/railanbaigazy/uade-api/internal/worker"
)

func main() {
	cfg := config.Load()

	metricsPort := os.Getenv("WORKER_METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9091"
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		log.Printf("worker: metrics listening on :%s/metrics", metricsPort)

		if err := http.ListenAndServe(":"+metricsPort, mux); err != nil {
			log.Printf("worker: metrics server stopped: %v", err)
		}
	}()

	db, err := sqlx.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatal("worker: failed to connect to DB:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("worker: database not reachable:", err)
	}

	mq, err := rabbitmq.Connect(cfg.AMQPURL)
	if err != nil {
		log.Fatal("worker: failed to connect to RabbitMQ:", err)
	}
	defer mq.Close()

	if err := rabbitmq.DeclareTopology(mq.Ch); err != nil {
		log.Fatal("worker: failed to declare RabbitMQ topology:", err)
	}

	gen := contracts.NewGenerator("contracts")

	c := worker.NewContractConsumer(db, mq.Ch, gen)

	log.Println("worker: waiting for messages...")
	if err := c.Run(); err != nil {
		log.Fatal("worker stopped:", err)
	}
}
