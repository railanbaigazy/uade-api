package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/railanbaigazy/uade-api/internal/contracts"
	"github.com/railanbaigazy/uade-api/internal/observability"
	"github.com/railanbaigazy/uade-api/internal/rabbitmq"
)

type ContractConsumer struct {
	db  *sqlx.DB
	ch  *amqp.Channel
	gen *contracts.Generator
}

func NewContractConsumer(db *sqlx.DB, ch *amqp.Channel, gen *contracts.Generator) *ContractConsumer {
	return &ContractConsumer{db: db, ch: ch, gen: gen}
}

type generateMsg struct {
	AgreementID string `json:"agreement_id"`
}

func (c *ContractConsumer) Run() error {
	if err := c.ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	msgs, err := c.ch.Consume(
		rabbitmq.QueueGenerateContract,
		"contract-worker",
		false, // autoAck=false
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
			start := time.Now()

			err := c.handleDelivery(d)
			if err != nil {
				requeue := shouldRequeue(err)

				log.Printf("worker: consume failed agreement_id=%s err=%v action=nack requeue=%v",
					extractAgreementID(d.Body), err, requeue)

				observability.MQConsumeTotal.WithLabelValues(rabbitmq.QueueGenerateContract, "error").Inc()
				observability.MQNackTotal.WithLabelValues(rabbitmq.QueueGenerateContract, boolLabel(requeue)).Inc()
				observability.MQConsumeDurationSeconds.WithLabelValues(rabbitmq.QueueGenerateContract).
					Observe(time.Since(start).Seconds())

				_ = d.Nack(false, requeue)
				continue
			}

			log.Printf("worker: consume ok action=ack duration_ms=%d",
				time.Since(start).Milliseconds())

			observability.MQConsumeTotal.WithLabelValues(rabbitmq.QueueGenerateContract, "ok").Inc()
			observability.MQAckTotal.WithLabelValues(rabbitmq.QueueGenerateContract).Inc()
			observability.MQConsumeDurationSeconds.WithLabelValues(rabbitmq.QueueGenerateContract).
				Observe(time.Since(start).Seconds())

			_ = d.Ack(false)
		}
	}()

	<-forever
	return nil
}

func (c *ContractConsumer) handleDelivery(d amqp.Delivery) error {
	var m generateMsg
	if err := json.Unmarshal(d.Body, &m); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	if m.AgreementID == "" {
		return fmt.Errorf("empty agreement_id")
	}

	log.Printf("worker: received agreement_id=%s delivery_tag=%d", m.AgreementID, d.DeliveryTag)

	var agreement models.Agreement
	err := c.db.Get(&agreement, "SELECT * FROM agreements WHERE id=$1", m.AgreementID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("agreement not found: %s", m.AgreementID)
		}
		return fmt.Errorf("db get agreement: %w", err)
	}

	if agreement.Status != "active" {
		return fmt.Errorf("agreement %s status=%s not active", m.AgreementID, agreement.Status)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	contractURL, contractHash, err := c.gen.Generate(ctx, &agreement)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	_, err = c.db.Exec(`
		UPDATE agreements 
		SET contract_url = $1, contract_hash = $2
		WHERE id = $3
	`, contractURL, contractHash, m.AgreementID)
	if err != nil {
		return fmt.Errorf("db update contract: %w", err)
	}

	log.Printf("worker: contract generated agreement_id=%s url=%s", m.AgreementID, contractURL)
	return nil
}

// shouldRequeue: transient errors -> true, permanent errors -> false
func shouldRequeue(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()

	// permanent: message is bad / cannot succeed later
	if strings.Contains(s, "unmarshal:") ||
		strings.Contains(s, "empty agreement_id") ||
		strings.Contains(s, "agreement not found") ||
		strings.Contains(s, "not active") {
		return false
	}

	// transient: DB down, file system, generator, etc.
	return true
}

func boolLabel(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// best-effort only for logging
func extractAgreementID(body []byte) string {
	var m generateMsg
	if err := json.Unmarshal(body, &m); err != nil {
		return ""
	}
	return m.AgreementID
}
