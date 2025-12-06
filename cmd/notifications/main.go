package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/railanbaigazy/uade-api/internal/config"
)

func main() {
	log.Println("[notifications] starting...")

	cfg := config.Load()

	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("[notifications] failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("[notifications] failed to open channel: %v", err)
	}
	defer ch.Close()

	queueName := "uade-notifications"

	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatalf("[notifications] failed to declare queue: %v", err)
	}

	if err := ch.QueueBind(
		q.Name,
		"agreement.*",      // routing key pattern
		cfg.RabbitExchange, // exchange
		false,
		nil,
	); err != nil {
		log.Fatalf("[notifications] failed to bind queue: %v", err)
	}

	log.Printf("[notifications] waiting for messages in queue=%s binding=%s -> %s",
		q.Name, cfg.RabbitExchange, "agreement.*")

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,  // auto-ack
		false, // exclusive
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[notifications] failed to register consumer: %v", err)
	}

	// 5. Обрабатываем сообщениxя
	go func() {
		for d := range msgs {
			var payload map[string]any
			if err := json.Unmarshal(d.Body, &payload); err != nil {
				log.Printf("[notifications] %s invalid json: %s", d.RoutingKey, string(d.Body))
				continue
			}
			log.Printf("[notifications] event=%s payload=%+v", d.RoutingKey, payload)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("[notifications] shutting down...")
}
