package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/railanbaigazy/uade-api/internal/observability"
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

type NotificationEvent struct {
	Type        string                 `json:"type"`
	UserID      int64                  `json:"user_id"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	AgreementID *int                   `json:"agreement_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (p *Publisher) PublishGenerateContract(ctx context.Context, agreementID string) error {
	body, err := json.Marshal(GenerateContractMessage{AgreementID: agreementID})
	if err != nil {
		observability.MQPublishTotal.WithLabelValues(ExchangeAgreements, RoutingGenerateContract, "error").Inc()
		log.Printf("mq publish failed: marshal error agreement_id=%s err=%v", agreementID, err)
		return fmt.Errorf("marshal message: %w", err)
	}

	err = p.ch.PublishWithContext(
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
	if err != nil {
		observability.MQPublishTotal.WithLabelValues(ExchangeAgreements, RoutingGenerateContract, "error").Inc()
		log.Printf("mq publish failed: agreement_id=%s exchange=%s key=%s err=%v",
			agreementID, ExchangeAgreements, RoutingGenerateContract, err)
		return err
	}

	observability.MQPublishTotal.WithLabelValues(ExchangeAgreements, RoutingGenerateContract, "ok").Inc()
	log.Printf("mq publish ok: agreement_id=%s exchange=%s key=%s", agreementID, ExchangeAgreements, RoutingGenerateContract)
	return nil
}

func (p *Publisher) PublishAgreementAccepted(ctx context.Context, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal notification event: %w", err)
	}

	return p.ch.PublishWithContext(
		ctx,
		ExchangeNotifications,
		RoutingAgreementAccepted,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func (p *Publisher) PublishPaymentReminder(ctx context.Context, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal notification event: %w", err)
	}

	return p.ch.PublishWithContext(
		ctx,
		ExchangeNotifications,
		RoutingPaymentReminder,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func (p *Publisher) PublishNotification(ctx context.Context, routingKey string, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal notification event: %w", err)
	}

	return p.ch.PublishWithContext(
		ctx,
		ExchangeNotifications,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}
