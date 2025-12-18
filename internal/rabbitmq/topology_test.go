package rabbitmq

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestDeclareTopology(t *testing.T) {
	// Skip if RabbitMQ is not available
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Skip("RabbitMQ not available, skipping integration test")
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatal("Failed to open channel:", err)
	}
	defer ch.Close()

	t.Run("successfully declare topology", func(t *testing.T) {
		err := DeclareTopology(ch)
		assert.NoError(t, err)

		// Verify exchanges exist by declaring them again (idempotent)
		err = ch.ExchangeDeclare(
			ExchangeAgreements,
			"direct",
			true,
			false,
			false,
			false,
			nil,
		)
		assert.NoError(t, err)

		err = ch.ExchangeDeclare(
			ExchangeNotifications,
			"topic",
			true,
			false,
			false,
			false,
			nil,
		)
		assert.NoError(t, err)

		// Verify queues exist
		_, err = ch.QueueDeclarePassive(
			QueueGenerateContract,
			true,
			false,
			false,
			false,
			nil,
		)
		assert.NoError(t, err)

		_, err = ch.QueueDeclarePassive(
			QueueNotifications,
			true,
			false,
			false,
			false,
			nil,
		)
		assert.NoError(t, err)

		_, err = ch.QueueDeclarePassive(
			QueueNotificationsDLQ,
			true,
			false,
			false,
			false,
			nil,
		)
		assert.NoError(t, err)
	})

	t.Run("topology is idempotent", func(t *testing.T) {
		// Should be able to declare multiple times without error
		err := DeclareTopology(ch)
		assert.NoError(t, err)

		err = DeclareTopology(ch)
		assert.NoError(t, err)
	})
}

func TestConnect(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		conn, err := Connect("amqp://guest:guest@localhost:5672/")
		if err != nil {
			t.Skip("RabbitMQ not available, skipping test")
			return
		}
		defer conn.Close()

		assert.NotNil(t, conn)
		assert.NotNil(t, conn.Conn)
		assert.NotNil(t, conn.Ch)
	})

	t.Run("failed connection", func(t *testing.T) {
		conn, err := Connect("amqp://invalid:invalid@localhost:9999/")
		assert.Error(t, err)
		assert.Nil(t, conn)
	})
}

func TestConn_Close(t *testing.T) {
	t.Run("close valid connection", func(t *testing.T) {
		conn, err := Connect("amqp://guest:guest@localhost:5672/")
		if err != nil {
			t.Skip("RabbitMQ not available, skipping test")
			return
		}

		err = conn.Close()
		assert.NoError(t, err)
	})

	t.Run("close nil connection", func(t *testing.T) {
		var conn *Conn
		err := conn.Close()
		assert.NoError(t, err)
	})
}

func TestRoutingConstants(t *testing.T) {
	t.Run("verify routing key constants", func(t *testing.T) {
		assert.Equal(t, "agreements.events", ExchangeAgreements)
		assert.Equal(t, "notifications.events", ExchangeNotifications)
		assert.Equal(t, "agreements.contract.generate", QueueGenerateContract)
		assert.Equal(t, "notifications.process", QueueNotifications)
		assert.Equal(t, "notifications.process.dlq", QueueNotificationsDLQ)
		assert.Equal(t, "agreements.generate_contract", RoutingGenerateContract)
		assert.Equal(t, "agreements.accepted", RoutingAgreementAccepted)
		assert.Equal(t, "notifications.payment_reminder", RoutingPaymentReminder)
		assert.Equal(t, "notifications.#", RoutingNotificationAll)
	})
}
