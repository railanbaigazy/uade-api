package rabbitmq

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func setupTestRabbitMQ(t *testing.T) (*amqp.Connection, *amqp.Channel) {
	// Skip if RabbitMQ is not available
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Skip("RabbitMQ not available, skipping integration test")
		return nil, nil
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		t.Fatal("Failed to open channel:", err)
	}

	// Declare test topology
	if err := DeclareTopology(ch); err != nil {
		ch.Close()
		conn.Close()
		t.Fatal("Failed to declare topology:", err)
	}

	return conn, ch
}

func TestPublisher_PublishAgreementAccepted(t *testing.T) {
	conn, ch := setupTestRabbitMQ(t)
	if conn == nil {
		return
	}
	defer conn.Close()
	defer ch.Close()

	publisher := NewPublisher(ch)

	t.Run("successfully publish agreement accepted event", func(t *testing.T) {
		agreementID := 123
		event := NotificationEvent{
			Type:        "agreement_accepted",
			UserID:      1,
			Title:       "Agreement Accepted",
			Message:     "Your agreement has been accepted",
			AgreementID: &agreementID,
			Metadata: map[string]interface{}{
				"amount":   1000.0,
				"currency": "KZT",
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := publisher.PublishAgreementAccepted(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("publish with context cancellation", func(t *testing.T) {
		agreementID := 456
		event := NotificationEvent{
			Type:        "agreement_accepted",
			UserID:      2,
			Title:       "Agreement Accepted",
			Message:     "Your agreement has been accepted",
			AgreementID: &agreementID,
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := publisher.PublishAgreementAccepted(ctx, event)
		assert.Error(t, err)
	})
}

func TestPublisher_PublishPaymentReminder(t *testing.T) {
	conn, ch := setupTestRabbitMQ(t)
	if conn == nil {
		return
	}
	defer conn.Close()
	defer ch.Close()

	publisher := NewPublisher(ch)

	t.Run("successfully publish payment reminder", func(t *testing.T) {
		agreementID := 789
		event := NotificationEvent{
			Type:        "payment_reminder",
			UserID:      3,
			Title:       "Payment Reminder",
			Message:     "Your payment is due soon",
			AgreementID: &agreementID,
			Metadata: map[string]interface{}{
				"due_date": "2024-12-25",
				"amount":   500.0,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := publisher.PublishPaymentReminder(ctx, event)
		assert.NoError(t, err)
	})
}

func TestPublisher_PublishGenerateContract(t *testing.T) {
	conn, ch := setupTestRabbitMQ(t)
	if conn == nil {
		return
	}
	defer conn.Close()
	defer ch.Close()

	publisher := NewPublisher(ch)

	t.Run("successfully publish contract generation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := publisher.PublishGenerateContract(ctx, "123")
		assert.NoError(t, err)
	})
}

func TestNotificationEventSerialization(t *testing.T) {
	t.Run("serialize event with all fields", func(t *testing.T) {
		agreementID := 123
		event := NotificationEvent{
			Type:        "agreement_accepted",
			UserID:      1,
			Title:       "Test Title",
			Message:     "Test Message",
			AgreementID: &agreementID,
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
		}

		data, err := json.Marshal(event)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		var decoded NotificationEvent
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, event.Type, decoded.Type)
		assert.Equal(t, event.UserID, decoded.UserID)
		assert.Equal(t, event.Title, decoded.Title)
		assert.Equal(t, *event.AgreementID, *decoded.AgreementID)
	})

	t.Run("serialize event without optional fields", func(t *testing.T) {
		event := NotificationEvent{
			Type:    "payment_reminder",
			UserID:  2,
			Title:   "Reminder",
			Message: "Pay soon",
		}

		data, err := json.Marshal(event)
		assert.NoError(t, err)

		var decoded NotificationEvent
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, event.Type, decoded.Type)
		assert.Nil(t, decoded.AgreementID)
		assert.Nil(t, decoded.Metadata)
	})
}
