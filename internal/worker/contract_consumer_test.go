package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/stretchr/testify/assert"
)

type stubGenerator struct {
	path string
	hash string
	err  error
}

func (g *stubGenerator) Generate(_ context.Context, _ *models.Agreement) (string, string, error) {
	if g.err != nil {
		return "", "", g.err
	}
	return g.path, g.hash, nil
}

func newConsumer(t *testing.T, gen ContractGenerator) (*ContractConsumer, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	consumer := NewContractConsumer(sqlxDB, nil, gen)
	return consumer, mock
}

func agreementRows(id int, status string, now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id", "principal_amount", "interest_rate",
		"total_amount", "currency", "created_at", "accepted_at", "disbursed_at", "start_date",
		"due_date", "completed_at", "payment_frequency", "number_of_payments", "status",
		"contract_url", "contract_hash",
	}).AddRow(
		id, int64(10), int64(20), 30, 1000.0, 0.1, 1100.0, "KZT", now, nil, nil, now, now.Add(24*time.Hour),
		nil, "monthly", 12, status, nil, nil,
	)
}

func TestContractConsumer_handleDelivery(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		gen := &stubGenerator{
			path: "/tmp/contract.pdf",
			hash: "hash",
		}
		consumer, mock := newConsumer(t, gen)

		mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
			WithArgs("123").
			WillReturnRows(agreementRows(123, "active", now))

		mock.ExpectExec(`UPDATE agreements\s+SET contract_url`).
			WithArgs(gen.path, gen.hash, "123").
			WillReturnResult(sqlmock.NewResult(0, 1))

		body, _ := json.Marshal(generateMsg{AgreementID: "123"})
		err := consumer.handleDelivery(amqp.Delivery{Body: body})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid json", func(t *testing.T) {
		consumer, mock := newConsumer(t, &stubGenerator{})

		err := consumer.handleDelivery(amqp.Delivery{Body: []byte("bad json")})
		assert.ErrorContains(t, err, "unmarshal")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty agreement id", func(t *testing.T) {
		consumer, mock := newConsumer(t, &stubGenerator{})

		body, _ := json.Marshal(generateMsg{})
		err := consumer.handleDelivery(amqp.Delivery{Body: body})
		assert.ErrorContains(t, err, "empty agreement_id")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("agreement not found", func(t *testing.T) {
		consumer, mock := newConsumer(t, &stubGenerator{})

		mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
			WithArgs("404").
			WillReturnError(sql.ErrNoRows)

		body, _ := json.Marshal(generateMsg{AgreementID: "404"})
		err := consumer.handleDelivery(amqp.Delivery{Body: body})
		assert.ErrorContains(t, err, "agreement not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("agreement not active", func(t *testing.T) {
		consumer, mock := newConsumer(t, &stubGenerator{})

		mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
			WithArgs("2").
			WillReturnRows(agreementRows(2, "cancelled", now))

		body, _ := json.Marshal(generateMsg{AgreementID: "2"})
		err := consumer.handleDelivery(amqp.Delivery{Body: body})
		assert.ErrorContains(t, err, "not active")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db query failure", func(t *testing.T) {
		consumer, mock := newConsumer(t, &stubGenerator{})

		mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
			WithArgs("500").
			WillReturnError(errors.New("db down"))

		body, _ := json.Marshal(generateMsg{AgreementID: "500"})
		err := consumer.handleDelivery(amqp.Delivery{Body: body})
		assert.ErrorContains(t, err, "db get agreement")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("generator failure bubbles", func(t *testing.T) {
		brokenGen := &stubGenerator{err: errors.New("cannot generate")}
		consumer, mock := newConsumer(t, brokenGen)

		mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
			WithArgs("1").
			WillReturnRows(agreementRows(1, "active", now))

		body, _ := json.Marshal(generateMsg{AgreementID: "1"})
		err := consumer.handleDelivery(amqp.Delivery{Body: body})
		assert.ErrorContains(t, err, "generate")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update failure bubbles", func(t *testing.T) {
		gen := &stubGenerator{
			path: "/tmp/contract.pdf",
			hash: "hash",
		}
		consumer, mock := newConsumer(t, gen)

		mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
			WithArgs("3").
			WillReturnRows(agreementRows(3, "active", now))

		mock.ExpectExec(`UPDATE agreements\s+SET contract_url`).
			WithArgs(gen.path, gen.hash, "3").
			WillReturnError(errors.New("write failed"))

		body, _ := json.Marshal(generateMsg{AgreementID: "3"})
		err := consumer.handleDelivery(amqp.Delivery{Body: body})
		assert.ErrorContains(t, err, "db update contract")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestShouldRequeue(t *testing.T) {
	assert.False(t, shouldRequeue(nil))
	assert.False(t, shouldRequeue(errors.New("unmarshal: bad")))
	assert.False(t, shouldRequeue(errors.New("empty agreement_id")))
	assert.False(t, shouldRequeue(errors.New("agreement not found")))
	assert.False(t, shouldRequeue(errors.New("not active")))

	assert.True(t, shouldRequeue(errors.New("db failure")))
	assert.True(t, shouldRequeue(errors.New("generate: timeout")))
}

func TestExtractAgreementID(t *testing.T) {
	body, _ := json.Marshal(generateMsg{AgreementID: "123"})
	assert.Equal(t, "123", extractAgreementID(body))

	assert.Equal(t, "", extractAgreementID([]byte("not json")))
}
