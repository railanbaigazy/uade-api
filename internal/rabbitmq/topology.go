package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// exchange
	ExchangeAgreements = "agreements.events"

	// queue
	QueueGenerateContract = "agreements.contract.generate"

	// routing key
	RoutingGenerateContract = "agreements.generate_contract"
)

func DeclareTopology(ch *amqp.Channel) error {
	// 1. Exchange
	if err := ch.ExchangeDeclare(
		ExchangeAgreements,
		"direct",
		true,  // durable
		false, // auto-delete
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	// 2. Queue
	q, err := ch.QueueDeclare(
		QueueGenerateContract,
		true, // durable
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	// 3. Binding
	if err := ch.QueueBind(
		q.Name,
		RoutingGenerateContract,
		ExchangeAgreements,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}

	return nil
}
