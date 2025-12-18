package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/railanbaigazy/uade-api/internal/rabbitmq"
)

type NotificationConsumer struct {
	db *sqlx.DB
	ch *amqp.Channel
}

func NewNotificationConsumer(db *sqlx.DB, ch *amqp.Channel) *NotificationConsumer {
	return &NotificationConsumer{db: db, ch: ch}
}

type notificationMsg struct {
	Type        string                 `json:"type"`
	UserID      int64                  `json:"user_id"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	AgreementID *int                   `json:"agreement_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (c *NotificationConsumer) Run() error {
	// Set QoS to process one message at a time
	if err := c.ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	msgs, err := c.ch.Consume(
		rabbitmq.QueueNotifications,
		"notification-worker",
		false, // autoAck=false for manual acknowledgment
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	forever := make(chan struct{})

	go func() {
		for d := range msgs {
			if err := c.handleDelivery(d); err != nil {
				log.Printf("notification-worker: failed: %v", err)

				// Check retry count from headers
				retryCount := getRetryCount(d)
				maxRetries := 3

				if retryCount >= maxRetries {
					log.Printf("notification-worker: max retries reached, moving to DLQ")
					// Reject without requeue - will go to DLQ
					_ = d.Nack(false, false)
				} else {
					// Increment retry count and requeue with delay
					log.Printf("notification-worker: retry %d/%d", retryCount+1, maxRetries)
					_ = d.Nack(false, true)
				}
				continue
			}

			// Successfully processed
			_ = d.Ack(false)
		}
	}()

	log.Println("notification-worker: waiting for messages...")
	<-forever
	return nil
}

func (c *NotificationConsumer) handleDelivery(d amqp.Delivery) error {
	var m notificationMsg
	if err := json.Unmarshal(d.Body, &m); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	// Validate required fields
	if m.UserID == 0 {
		return fmt.Errorf("invalid user_id")
	}
	if m.Type == "" {
		return fmt.Errorf("empty notification type")
	}
	if m.Title == "" {
		return fmt.Errorf("empty notification title")
	}

	// Convert metadata to JSON string
	var metadataJSON *string
	if m.Metadata != nil {
		metadataBytes, err := json.Marshal(m.Metadata)
		if err != nil {
			log.Printf("warning: failed to marshal metadata: %v", err)
		} else {
			metadataStr := string(metadataBytes)
			metadataJSON = &metadataStr
		}
	}

	// Insert notification into database
	query := `
		INSERT INTO notifications (
			user_id, type, title, message, status, agreement_id, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var notificationID int64
	err := c.db.QueryRow(
		query,
		m.UserID,
		m.Type,
		m.Title,
		m.Message,
		models.NotificationStatusUnread,
		m.AgreementID,
		metadataJSON,
		time.Now(),
	).Scan(&notificationID)

	if err != nil {
		return fmt.Errorf("db insert notification: %w", err)
	}

	log.Printf("notification-worker: created notification id=%d for user=%d type=%s",
		notificationID, m.UserID, m.Type)
	return nil
}

func getRetryCount(d amqp.Delivery) int {
	if d.Headers == nil {
		return 0
	}
	if count, ok := d.Headers["x-retry-count"].(int32); ok {
		return int(count)
	}
	if count, ok := d.Headers["x-retry-count"].(int); ok {
		return count
	}
	return 0
}
