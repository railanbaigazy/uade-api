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

const (
	defaultMaxRetries  = 5
	defaultRetryDelay  = 2 * time.Second
	defaultConnTimeout = 10 * time.Second
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

// NewPublisher creates a new RabbitMQ publisher with retry logic
func NewPublisher(url, exchange string) (*AMQPPublisher, error) {
	return NewPublisherWithRetry(url, exchange, defaultMaxRetries, defaultRetryDelay, defaultConnTimeout)
}

// NewPublisherWithRetry creates a new RabbitMQ publisher with custom retry settings
func NewPublisherWithRetry(url, exchange string, maxRetries int, retryDelay, connTimeout time.Duration) (*AMQPPublisher, error) {
	var conn *amqp.Connection
	var ch *amqp.Channel
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[mq] retrying connection (attempt %d/%d) in %v...", attempt+1, maxRetries, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // exponential backoff
		}

		conn, err = amqp.Dial(url)

		if err != nil {
			log.Printf("[mq] connection attempt %d/%d failed: %v", attempt+1, maxRetries, err)
			continue
		}

		ch, err = conn.Channel()
		if err != nil {
			conn.Close()
			log.Printf("[mq] channel creation attempt %d/%d failed: %v", attempt+1, maxRetries, err)
			continue
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
			log.Printf("[mq] exchange declaration attempt %d/%d failed: %v", attempt+1, maxRetries, err)
			continue
		}

		log.Printf("[mq] publisher connected successfully (attempt %d/%d)", attempt+1, maxRetries)
		return &AMQPPublisher{
			conn:     conn,
			channel:  ch,
			exchange: exchange,
		}, nil
	}

	return nil, fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", maxRetries, err)
}

func (p *AMQPPublisher) Publish(ctx context.Context, routingKey string, payload any) error {
	if p == nil {
		metrics.MQPublishErrorsTotal.WithLabelValues(routingKey).Inc()
		return fmt.Errorf("publisher is nil")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		metrics.MQPublishErrorsTotal.WithLabelValues(routingKey).Inc()
		log.Printf("[mq] [ERROR] failed to marshal payload for routing_key=%s: %v", routingKey, err)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	startTime := time.Now()
	err = p.channel.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // Make messages persistent
			Timestamp:    time.Now(),
		},
	)
	duration := time.Since(startTime)

	if err != nil {
		metrics.MQPublishErrorsTotal.WithLabelValues(routingKey).Inc()
		log.Printf("[mq] [ERROR] failed to publish routing_key=%s size=%d duration=%v err=%v",
			routingKey, len(body), duration, err)
		return err
	}

	metrics.MQPublishedTotal.WithLabelValues(routingKey).Inc()
	log.Printf("[mq] [INFO] published routing_key=%s size=%d duration=%v",
		routingKey, len(body), duration)

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
