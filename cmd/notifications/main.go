package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	log.Println("[notifications] starting...")

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@rabbitmq:5672/"
	}

	exchange := os.Getenv("RABBITMQ_EXCHANGE")
	if exchange == "" {
		exchange = "uade.events"
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("[notifications] failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("[notifications] failed to open channel: %v", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		log.Fatalf("[notifications] failed to declare exchange: %v", err)
	}

	queueName := "uade-notifications"

	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[notifications] failed to declare queue: %v", err)
	}

	if err := ch.QueueBind(
		q.Name,
		"agreement.*",
		exchange,
		false,
		nil,
	); err != nil {
		log.Fatalf("[notifications] failed to bind queue: %v", err)
	}

	log.Printf("[notifications] waiting for messages in queue=%s, exchange=%s", q.Name, exchange)

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[notifications] failed to register consumer: %v", err)
	}

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
