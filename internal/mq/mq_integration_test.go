package mq_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/railanbaigazy/uade-api/internal/mq"
)

func TestIntegration_PublishAndConsume(t *testing.T) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	exchange := os.Getenv("RABBITMQ_EXCHANGE")
	if exchange == "" {
		exchange = "uade.events"
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		t.Skipf("RabbitMQ not available: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"uade_mq_test_queue",
		false, // durable
		true,  // auto-delete
		false,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to declare queue: %v", err)
	}

	routingKey := "test.mq.integration"

	if err := ch.QueueBind(q.Name, routingKey, exchange, false, nil); err != nil {
		t.Fatalf("failed to bind queue: %v", err)
	}

	pub, err := mq.NewPublisher(url, exchange)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	defer pub.Close()

	payload := map[string]any{"hello": "world", "n": 42}

	if err := pub.Publish(context.Background(), routingKey, payload); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	var msg amqp.Delivery
	ok := false

	for i := 0; i < 10; i++ {
		msg, ok, err = ch.Get(q.Name, true)
		if err != nil {
			t.Fatalf("failed to get message: %v", err)
		}
		if ok {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !ok {
		t.Fatalf("no message received from queue")
	}

	var got map[string]any
	if err := json.Unmarshal(msg.Body, &got); err != nil {
		t.Fatalf("invalid json in message: %v", err)
	}

	if got["hello"] != "world" {
		t.Fatalf("unexpected payload: %+v", got)
	}
}
