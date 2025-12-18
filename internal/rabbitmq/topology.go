package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// exchanges
	ExchangeAgreements    = "agreements.events"
	ExchangeNotifications = "notifications.events"

	// queues - contracts
	QueueGenerateContract = "agreements.contract.generate"

	// queues - notifications
	QueueNotifications    = "notifications.process"
	QueueNotificationsDLQ = "notifications.process.dlq"

	// routing keys - contracts
	RoutingGenerateContract = "agreements.generate_contract"

	// routing keys - notifications
	RoutingAgreementAccepted = "agreements.accepted"
	RoutingPaymentReminder   = "notifications.payment_reminder"
	RoutingNotificationAll   = "notifications.#"
)

func DeclareTopology(ch *amqp.Channel) error {
	// 1. Declare Exchanges
	if err := ch.ExchangeDeclare(
		ExchangeAgreements,
		"direct",
		true,  // durable
		false, // auto-delete
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange agreements: %w", err)
	}

	if err := ch.ExchangeDeclare(
		ExchangeNotifications,
		"topic", // topic exchange for flexible routing
		true,    // durable
		false,   // auto-delete
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange notifications: %w", err)
	}

	// 2. Declare Contract Queue
	qContract, err := ch.QueueDeclare(
		QueueGenerateContract,
		true, // durable
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare queue contract: %w", err)
	}

	// 3. Bind Contract Queue
	if err := ch.QueueBind(
		qContract.Name,
		RoutingGenerateContract,
		ExchangeAgreements,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind contract queue: %w", err)
	}

	// 4. Declare Dead Letter Queue for Notifications
	qDLQ, err := ch.QueueDeclare(
		QueueNotificationsDLQ,
		true, // durable
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare DLQ: %w", err)
	}

	// 5. Declare Notifications Queue with DLQ configuration
	qNotifications, err := ch.QueueDeclare(
		QueueNotifications,
		true, // durable
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    ExchangeNotifications,
			"x-dead-letter-routing-key": QueueNotificationsDLQ,
		},
	)
	if err != nil {
		return fmt.Errorf("declare notifications queue: %w", err)
	}

	// 6. Bind Notifications Queue
	if err := ch.QueueBind(
		qNotifications.Name,
		RoutingNotificationAll,
		ExchangeNotifications,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind notifications queue: %w", err)
	}

	// 7. Bind DLQ to itself for dead letters
	if err := ch.QueueBind(
		qDLQ.Name,
		QueueNotificationsDLQ,
		ExchangeNotifications,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind DLQ: %w", err)
	}

	return nil
}
