package worker

import (
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/stretchr/testify/assert"
)

func TestNotificationConsumer_handleDelivery(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	t.Run("successfully handle notification message", func(t *testing.T) {
		consumer := NewNotificationConsumer(sqlxDB, nil)

		agreementID := 123
		msg := notificationMsg{
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

		body, err := json.Marshal(msg)
		assert.NoError(t, err)

		delivery := amqp.Delivery{
			Body: body,
		}

		mock.ExpectQuery("INSERT INTO notifications").
			WithArgs(
				msg.UserID,
				msg.Type,
				msg.Title,
				msg.Message,
				models.NotificationStatusUnread,
				msg.AgreementID,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err = consumer.handleDelivery(delivery)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handle message with missing user_id", func(t *testing.T) {
		consumer := NewNotificationConsumer(sqlxDB, nil)

		msg := notificationMsg{
			Type:    "agreement_accepted",
			Title:   "Test",
			Message: "Test message",
		}

		body, err := json.Marshal(msg)
		assert.NoError(t, err)

		delivery := amqp.Delivery{
			Body: body,
		}

		err = consumer.handleDelivery(delivery)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user_id")
	})

	t.Run("handle message with missing type", func(t *testing.T) {
		consumer := NewNotificationConsumer(sqlxDB, nil)

		msg := notificationMsg{
			UserID:  1,
			Title:   "Test",
			Message: "Test message",
		}

		body, err := json.Marshal(msg)
		assert.NoError(t, err)

		delivery := amqp.Delivery{
			Body: body,
		}

		err = consumer.handleDelivery(delivery)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty notification type")
	})

	t.Run("handle message with invalid JSON", func(t *testing.T) {
		consumer := NewNotificationConsumer(sqlxDB, nil)

		delivery := amqp.Delivery{
			Body: []byte("invalid json"),
		}

		err = consumer.handleDelivery(delivery)
		assert.Error(t, err)
	})

	t.Run("handle message without metadata", func(t *testing.T) {
		consumer := NewNotificationConsumer(sqlxDB, nil)

		msg := notificationMsg{
			Type:    "payment_reminder",
			UserID:  2,
			Title:   "Payment Reminder",
			Message: "Your payment is due",
		}

		body, err := json.Marshal(msg)
		assert.NoError(t, err)

		delivery := amqp.Delivery{
			Body: body,
		}

		mock.ExpectQuery("INSERT INTO notifications").
			WithArgs(
				msg.UserID,
				msg.Type,
				msg.Title,
				msg.Message,
				models.NotificationStatusUnread,
				msg.AgreementID,
				nil,
				sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

		err = consumer.handleDelivery(delivery)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetRetryCount(t *testing.T) {
	t.Run("no headers", func(t *testing.T) {
		delivery := amqp.Delivery{}
		count := getRetryCount(delivery)
		assert.Equal(t, 0, count)
	})

	t.Run("with retry count as int32", func(t *testing.T) {
		delivery := amqp.Delivery{
			Headers: amqp.Table{
				"x-retry-count": int32(3),
			},
		}
		count := getRetryCount(delivery)
		assert.Equal(t, 3, count)
	})

	t.Run("with retry count as int", func(t *testing.T) {
		delivery := amqp.Delivery{
			Headers: amqp.Table{
				"x-retry-count": 5,
			},
		}
		count := getRetryCount(delivery)
		assert.Equal(t, 5, count)
	})

	t.Run("without retry count header", func(t *testing.T) {
		delivery := amqp.Delivery{
			Headers: amqp.Table{
				"other-header": "value",
			},
		}
		count := getRetryCount(delivery)
		assert.Equal(t, 0, count)
	})
}

func TestNotificationConsumer_DatabaseFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	consumer := NewNotificationConsumer(sqlxDB, nil)

	t.Run("database insert failure", func(t *testing.T) {
		msg := notificationMsg{
			Type:    "agreement_accepted",
			UserID:  1,
			Title:   "Test",
			Message: "Test message",
		}

		body, err := json.Marshal(msg)
		assert.NoError(t, err)

		delivery := amqp.Delivery{
			Body: body,
		}

		mock.ExpectQuery("INSERT INTO notifications").
			WillReturnError(assert.AnError)

		err = consumer.handleDelivery(delivery)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db insert notification")
	})
}

func TestNotificationConsumer_MetadataHandling(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	consumer := NewNotificationConsumer(sqlxDB, nil)

	t.Run("handle complex metadata", func(t *testing.T) {
		msg := notificationMsg{
			Type:    "payment_overdue",
			UserID:  1,
			Title:   "Payment Overdue",
			Message: "Your payment is overdue",
			Metadata: map[string]interface{}{
				"amount":       500.75,
				"currency":     "KZT",
				"days_overdue": 5,
				"penalties": map[string]interface{}{
					"late_fee": 50.0,
					"interest": 10.5,
				},
			},
		}

		body, err := json.Marshal(msg)
		assert.NoError(t, err)

		delivery := amqp.Delivery{
			Body: body,
		}

		mock.ExpectQuery("INSERT INTO notifications").
			WithArgs(
				msg.UserID,
				msg.Type,
				msg.Title,
				msg.Message,
				models.NotificationStatusUnread,
				msg.AgreementID,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err = consumer.handleDelivery(delivery)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func BenchmarkNotificationConsumer_handleDelivery(b *testing.B) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	consumer := NewNotificationConsumer(sqlxDB, nil)

	agreementID := 123
	msg := notificationMsg{
		Type:        "agreement_accepted",
		UserID:      1,
		Title:       "Agreement Accepted",
		Message:     "Your agreement has been accepted",
		AgreementID: &agreementID,
	}

	body, _ := json.Marshal(msg)

	for i := 0; i < b.N; i++ {
		delivery := amqp.Delivery{
			Body: body,
		}

		mock.ExpectQuery("INSERT INTO notifications").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(i)))

		_ = consumer.handleDelivery(delivery)
	}
}
