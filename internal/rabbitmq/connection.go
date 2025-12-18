package rabbitmq

import (
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Conn struct {
	Conn *amqp.Connection
	Ch   *amqp.Channel
}

func Connect(amqpURL string) (*Conn, error) {
	var (
		conn *amqp.Connection
		ch   *amqp.Channel
		err  error
	)

	// retry logic
	for i := 1; i <= 10; i++ {
		conn, err = amqp.Dial(amqpURL)
		if err == nil {
			break
		}

		log.Printf("[rabbitmq] attempt %d/10 failed: %v", i, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ after retries: %w", err)
	}

	ch, err = conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	return &Conn{Conn: conn, Ch: ch}, nil
}

func (c *Conn) Close() error {
	if c == nil {
		return nil
	}
	if c.Ch != nil {
		_ = c.Ch.Close()
	}
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}
