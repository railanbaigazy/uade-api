package main

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/railanbaigazy/uade-api/internal/contracts"
	"github.com/railanbaigazy/uade-api/internal/rabbitmq"
	"github.com/railanbaigazy/uade-api/internal/worker"
)

func main() {
	cfg := config.Load()

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
