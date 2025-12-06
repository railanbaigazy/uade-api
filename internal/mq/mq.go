package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/railanbaigazy/uade-api/internal/metrics"
)

type Publisher interface {
	Publish(ctx context.Context, routingKey string, payload any) error
	Close() error
}

type AMQPPublisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

func NewPublisher(url, exchange string) (*AMQPPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		exchange,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &AMQPPublisher{
		conn:     conn,
		channel:  ch,
		exchange: exchange,
	}, nil
}

func (p *AMQPPublisher) Publish(ctx context.Context, routingKey string, payload any) error {
	if p == nil {
		return fmt.Errorf("publisher is nil")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		metrics.MQPublishErrorsTotal.WithLabelValues(routingKey).Inc()
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		metrics.MQPublishErrorsTotal.WithLabelValues(routingKey).Inc()
		log.Printf("[mq] failed to publish routing_key=%s err=%v", routingKey, err)
		return err
	}

	metrics.MQPublishedTotal.WithLabelValues(routingKey).Inc()
	log.Printf("[mq] published routing_key=%s size=%d", routingKey, len(body))

	return nil
}

func (p *AMQPPublisher) Close() error {
	if p == nil {
		return nil
	}
	if err := p.channel.Close(); err != nil {
		_ = p.conn.Close()
		return err
	}
	return p.conn.Close()
}
