package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/railanbaigazy/uade-api/internal/contracts"
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
			if err := c.handleDelivery(d); err != nil {
				log.Printf("worker: failed: %v", err)

				_ = d.Nack(false, true)
				continue
			}

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

	log.Printf("worker: contract generated for agreement_id=%s", m.AgreementID)
	return nil
}
