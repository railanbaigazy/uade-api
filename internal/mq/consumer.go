package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// Queue names
	NotificationQueue = "notifications.queue"
	DLQ               = "notifications.dlq"

	// Routing keys
	RoutingKeyAgreementAccepted  = "agreement.accepted"
	RoutingKeyAgreementCreated   = "agreement.created"
	RoutingKeyAgreementCancelled = "agreement.cancelled"
	RoutingKeyPaymentReminder    = "notification.payment_reminder"
	RoutingKeyOverdueAlert       = "notification.overdue_alert"
)

type Consumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	db       *sqlx.DB
}

// NewConsumer creates a new RabbitMQ consumer with DLQ support
func NewConsumer(url, exchange string, db *sqlx.DB) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
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

	// Declare DLQ (standalone queue for dead-lettered messages)
	dlqArgs := amqp.Table{
		"x-message-ttl": int32(86400000), // 24 hours in milliseconds
	}
	if _, err := ch.QueueDeclare(
		DLQ,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		dlqArgs,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Declare main queue with DLQ arguments
	// When a message is rejected/nacked, it will be sent to the default exchange
	// with routing key = DLQ queue name
	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    "", // Use default exchange (empty string)
		"x-dead-letter-routing-key": DLQ,
	}
	if _, err := ch.QueueDeclare(
		NotificationQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		queueArgs,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange with routing keys
	routingKeys := []string{
		RoutingKeyAgreementAccepted,
		RoutingKeyAgreementCreated,
		RoutingKeyAgreementCancelled,
		RoutingKeyPaymentReminder,
		RoutingKeyOverdueAlert,
	}

	for _, routingKey := range routingKeys {
		if err := ch.QueueBind(
			NotificationQueue,
			routingKey,
			exchange,
			false,
			nil,
		); err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("failed to bind queue with routing key %s: %w", routingKey, err)
		}
	}

	// Set QoS to process one message at a time
	if err := ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &Consumer{
		conn:     conn,
		channel:  ch,
		exchange: exchange,
		db:       db,
	}, nil
}

// Start begins consuming messages and processing them
func (c *Consumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		NotificationQueue,
		"",    // consumer tag
		false, // auto-ack (we'll manually ack)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Println("Consumer started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer context cancelled, stopping...")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				log.Println("Message channel closed")
				return fmt.Errorf("message channel closed")
			}

			if err := c.handleMessage(msg); err != nil {
				log.Printf("Error handling message: %v", err)
				// Reject and requeue (will go to DLQ after max retries)
				_ = msg.Nack(false, true)
			} else {
				// Acknowledge successful processing
				if err := msg.Ack(false); err != nil {
					log.Printf("Error acknowledging message: %v", err)
				}
			}
		}
	}
}

// handleMessage processes a single message
func (c *Consumer) handleMessage(msg amqp.Delivery) error {
	log.Printf("Received message: routing key=%s, body=%s", msg.RoutingKey, string(msg.Body))

	switch msg.RoutingKey {
	case RoutingKeyAgreementAccepted:
		return c.handleAgreementAccepted(msg.Body)
	case RoutingKeyAgreementCreated:
		return c.handleAgreementCreated(msg.Body)
	case RoutingKeyAgreementCancelled:
		return c.handleAgreementCancelled(msg.Body)
	case RoutingKeyPaymentReminder:
		return c.handlePaymentReminder(msg.Body)
	case RoutingKeyOverdueAlert:
		return c.handleOverdueAlert(msg.Body)
	default:
		log.Printf("Unknown routing key: %s", msg.RoutingKey)
		return fmt.Errorf("unknown routing key: %s", msg.RoutingKey)
	}
}

func (c *Consumer) handleAgreementAccepted(body []byte) error {
	var event AgreementAcceptedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("failed to unmarshal agreement accepted event: %w", err)
	}

	// Create notifications for both lender and borrower
	metadata, _ := json.Marshal(map[string]interface{}{
		"agreement_id": event.AgreementID,
		"post_id":      event.PostID,
	})
	metadataStr := string(metadata)

	// Notification for borrower
	borrowerQuery := `
		INSERT INTO notifications (user_id, type, title, message, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, NOW())
	`
	borrowerTitle := "Agreement Accepted"
	borrowerMsg := fmt.Sprintf("Your agreement #%d has been accepted by the lender.", event.AgreementID)
	if _, err := c.db.Exec(borrowerQuery, event.BorrowerID, "agreement_accepted", borrowerTitle, borrowerMsg, metadataStr); err != nil {
		return fmt.Errorf("failed to create borrower notification: %w", err)
	}

	// Notification for lender
	lenderTitle := "Agreement Accepted"
	lenderMsg := fmt.Sprintf("You have accepted agreement #%d.", event.AgreementID)
	if _, err := c.db.Exec(borrowerQuery, event.LenderID, "agreement_accepted", lenderTitle, lenderMsg, metadataStr); err != nil {
		return fmt.Errorf("failed to create lender notification: %w", err)
	}

	log.Printf("Created notifications for agreement accepted: agreement_id=%d", event.AgreementID)
	return nil
}

func (c *Consumer) handleAgreementCreated(body []byte) error {
	var event AgreementCreatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("failed to unmarshal agreement created event: %w", err)
	}

	// Create notification for lender
	metadata, _ := json.Marshal(map[string]interface{}{
		"agreement_id": event.AgreementID,
		"post_id":      event.PostID,
	})
	metadataStr := string(metadata)

	query := `
		INSERT INTO notifications (user_id, type, title, message, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, NOW())
	`
	title := "New Agreement Request"
	msg := fmt.Sprintf("A new agreement request #%d has been created for your post.", event.AgreementID)
	if _, err := c.db.Exec(query, event.LenderID, "agreement_created", title, msg, metadataStr); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	log.Printf("Created notification for agreement created: agreement_id=%d", event.AgreementID)
	return nil
}

func (c *Consumer) handleAgreementCancelled(body []byte) error {
	var event AgreementCancelledEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("failed to unmarshal agreement cancelled event: %w", err)
	}

	metadata, _ := json.Marshal(map[string]interface{}{
		"agreement_id": event.AgreementID,
		"post_id":      event.PostID,
	})
	metadataStr := string(metadata)

	query := `
		INSERT INTO notifications (user_id, type, title, message, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, NOW())
	`

	// Notify both parties
	title := "Agreement Cancelled"
	msg := fmt.Sprintf("Agreement #%d has been cancelled.", event.AgreementID)

	// Notify lender
	if _, err := c.db.Exec(query, event.LenderID, "agreement_cancelled", title, msg, metadataStr); err != nil {
		return fmt.Errorf("failed to create lender notification: %w", err)
	}

	// Notify borrower
	if _, err := c.db.Exec(query, event.BorrowerID, "agreement_cancelled", title, msg, metadataStr); err != nil {
		return fmt.Errorf("failed to create borrower notification: %w", err)
	}

	log.Printf("Created notifications for agreement cancelled: agreement_id=%d", event.AgreementID)
	return nil
}

func (c *Consumer) handlePaymentReminder(body []byte) error {
	var event PaymentReminderEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("failed to unmarshal payment reminder event: %w", err)
	}

	metadata, _ := json.Marshal(map[string]interface{}{
		"agreement_id": event.AgreementID,
		"due_date":     event.DueDate,
		"amount":       event.Amount,
		"currency":     event.Currency,
	})
	metadataStr := string(metadata)

	query := `
		INSERT INTO notifications (user_id, type, title, message, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, NOW())
	`
	title := "Payment Reminder"
	msg := fmt.Sprintf("Reminder: Payment of %.2f %s for agreement #%d is due on %s.",
		event.Amount, event.Currency, event.AgreementID, event.DueDate.Format("2006-01-02"))

	if _, err := c.db.Exec(query, event.UserID, "payment_reminder", title, msg, metadataStr); err != nil {
		return fmt.Errorf("failed to create payment reminder notification: %w", err)
	}

	log.Printf("Created payment reminder notification: agreement_id=%d, user_id=%d", event.AgreementID, event.UserID)
	return nil
}

func (c *Consumer) handleOverdueAlert(body []byte) error {
	var event OverdueAlertEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("failed to unmarshal overdue alert event: %w", err)
	}

	metadata, _ := json.Marshal(map[string]interface{}{
		"agreement_id": event.AgreementID,
		"due_date":     event.DueDate,
		"amount":       event.Amount,
		"currency":     event.Currency,
		"days_overdue": event.DaysOverdue,
	})
	metadataStr := string(metadata)

	query := `
		INSERT INTO notifications (user_id, type, title, message, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, NOW())
	`
	title := "Overdue Payment Alert"
	msg := fmt.Sprintf("ALERT: Payment of %.2f %s for agreement #%d is %d days overdue (due date: %s).",
		event.Amount, event.Currency, event.AgreementID, event.DaysOverdue, event.DueDate.Format("2006-01-02"))

	if _, err := c.db.Exec(query, event.UserID, "overdue_alert", title, msg, metadataStr); err != nil {
		return fmt.Errorf("failed to create overdue alert notification: %w", err)
	}

	log.Printf("Created overdue alert notification: agreement_id=%d, user_id=%d, days_overdue=%d",
		event.AgreementID, event.UserID, event.DaysOverdue)
	return nil
}

// Close closes the consumer connection
func (c *Consumer) Close() error {
	if c == nil {
		return nil
	}
	if err := c.channel.Close(); err != nil {
		_ = c.conn.Close()
		return err
	}
	return c.conn.Close()
}
