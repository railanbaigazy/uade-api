package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	ch *amqp.Channel
}

func NewPublisher(ch *amqp.Channel) *Publisher {
	return &Publisher{ch: ch}
}

type GenerateContractMessage struct {
	AgreementID string `json:"agreement_id"`
}

func (p *Publisher) PublishGenerateContract(ctx context.Context, agreementID string) error {
	body, err := json.Marshal(GenerateContractMessage{AgreementID: agreementID})
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	return p.ch.PublishWithContext(
		ctx,
		ExchangeAgreements,
		RoutingGenerateContract,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}
