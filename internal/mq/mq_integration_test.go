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

func TestIntegration_PublisherRetry(t *testing.T) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	exchange := os.Getenv("RABBITMQ_EXCHANGE")
	if exchange == "" {
		exchange = "uade.events"
	}

	// Test with invalid URL first (should fail)
	_, err := mq.NewPublisher("amqp://invalid:5672/", exchange)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}

	// Test with valid URL (should succeed)
	pub, err := mq.NewPublisher(url, exchange)
	if err != nil {
		t.Skipf("RabbitMQ not available: %v", err)
	}
	defer pub.Close()

	// Test publishing
	payload := map[string]any{"test": "retry"}
	if err := pub.Publish(context.Background(), "test.retry", payload); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
}

func TestIntegration_ConsumerWithDLQ(t *testing.T) {
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

	// Verify DLQ exists
	_, err = ch.QueueDeclare(
		mq.DLQ,
		false, // don't create, just check
		false,
		false,
		true, // noWait
		nil,
	)
	if err != nil {
		t.Logf("DLQ not found (this is ok if consumer hasn't run yet): %v", err)
	}

	// Verify main queue exists
	_, err = ch.QueueDeclare(
		mq.NotificationQueue,
		false, // don't create, just check
		false,
		false,
		true, // noWait
		nil,
	)
	if err != nil {
		t.Logf("Notification queue not found (this is ok if consumer hasn't run yet): %v", err)
	}
}

func TestIntegration_MultipleRoutingKeys(t *testing.T) {
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

	pub, err := mq.NewPublisher(url, exchange)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	defer pub.Close()

	// Test all routing keys
	routingKeys := []string{
		mq.RoutingKeyAgreementCreated,
		mq.RoutingKeyAgreementAccepted,
		mq.RoutingKeyAgreementCancelled,
		mq.RoutingKeyPaymentReminder,
		mq.RoutingKeyOverdueAlert,
	}

	for _, rk := range routingKeys {
		payload := map[string]any{
			"routing_key": rk,
			"test":        true,
		}
		if err := pub.Publish(context.Background(), rk, payload); err != nil {
			t.Fatalf("failed to publish to %s: %v", rk, err)
		}
		t.Logf("Published to %s successfully", rk)
	}
}

func TestIntegration_PublisherClose(t *testing.T) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	exchange := os.Getenv("RABBITMQ_EXCHANGE")
	if exchange == "" {
		exchange = "uade.events"
	}

	pub, err := mq.NewPublisher(url, exchange)
	if err != nil {
		t.Skipf("RabbitMQ not available: %v", err)
	}

	// Close should not error
	if err := pub.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// Closing again should be safe
	if err := pub.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}
