package mq

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/utils"
	"github.com/stretchr/testify/require"
)

func setupTestConsumer(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	return utils.NewSQLXMock(t)
}

func TestHandleAgreementAccepted(t *testing.T) {
	db, mock := setupTestConsumer(t)

	// Note: We can't easily test the full consumer without a real RabbitMQ connection,
	// but we can test the event unmarshaling and SQL execution logic

	event := AgreementAcceptedEvent{
		AgreementID: 1,
		LenderID:    10,
		BorrowerID:  20,
		PostID:      5,
		Status:      "active",
		AcceptedAt:  timePtr(time.Now()),
		StartDate:   timePtr(time.Now()),
		DueDate:     time.Now().Add(30 * 24 * time.Hour),
	}

	body, err := json.Marshal(event)
	require.NoError(t, err)

	// Mock the two INSERT queries (for borrower and lender)
	metadata, _ := json.Marshal(map[string]interface{}{
		"agreement_id": event.AgreementID,
		"post_id":      event.PostID,
	})
	metadataStr := string(metadata)

	mock.ExpectExec(`INSERT INTO notifications`).
		WithArgs(event.BorrowerID, "agreement_accepted", "Agreement Accepted", sqlmock.AnyArg(), metadataStr).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO notifications`).
		WithArgs(event.LenderID, "agreement_accepted", "Agreement Accepted", sqlmock.AnyArg(), metadataStr).
		WillReturnResult(sqlmock.NewResult(2, 1))

	// Create a minimal consumer struct for testing
	consumer := &Consumer{db: db}

	err = consumer.handleAgreementAccepted(body)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandlePaymentReminder(t *testing.T) {
	db, mock := setupTestConsumer(t)

	event := PaymentReminderEvent{
		AgreementID: 1,
		UserID:      10,
		DueDate:     time.Now().Add(7 * 24 * time.Hour),
		Amount:      1000.50,
		Currency:    "KZT",
	}

	body, err := json.Marshal(event)
	require.NoError(t, err)

	metadata, _ := json.Marshal(map[string]interface{}{
		"agreement_id": event.AgreementID,
		"due_date":     event.DueDate,
		"amount":       event.Amount,
		"currency":     event.Currency,
	})
	metadataStr := string(metadata)

	mock.ExpectExec(`INSERT INTO notifications`).
		WithArgs(event.UserID, "payment_reminder", "Payment Reminder", sqlmock.AnyArg(), metadataStr).
		WillReturnResult(sqlmock.NewResult(1, 1))

	consumer := &Consumer{db: db}

	err = consumer.handlePaymentReminder(body)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleOverdueAlert(t *testing.T) {
	db, mock := setupTestConsumer(t)

	event := OverdueAlertEvent{
		AgreementID: 1,
		UserID:      10,
		DueDate:     time.Now().Add(-5 * 24 * time.Hour),
		Amount:      1000.50,
		Currency:    "KZT",
		DaysOverdue: 5,
	}

	body, err := json.Marshal(event)
	require.NoError(t, err)

	metadata, _ := json.Marshal(map[string]interface{}{
		"agreement_id": event.AgreementID,
		"due_date":     event.DueDate,
		"amount":       event.Amount,
		"currency":     event.Currency,
		"days_overdue": event.DaysOverdue,
	})
	metadataStr := string(metadata)

	mock.ExpectExec(`INSERT INTO notifications`).
		WithArgs(event.UserID, "overdue_alert", "Overdue Payment Alert", sqlmock.AnyArg(), metadataStr).
		WillReturnResult(sqlmock.NewResult(1, 1))

	consumer := &Consumer{db: db}

	err = consumer.handleOverdueAlert(body)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleAgreementCreated(t *testing.T) {
	db, mock := setupTestConsumer(t)

	event := AgreementCreatedEvent{
		AgreementID:      1,
		LenderID:         10,
		BorrowerID:       20,
		PostID:           5,
		PrincipalAmount:  1000.0,
		InterestRate:     0.1,
		TotalAmount:      1100.0,
		Currency:         "KZT",
		DueDate:          time.Now().Add(30 * 24 * time.Hour),
		PaymentFrequency: "monthly",
		NumberOfPayments: 3,
		Status:           "pending",
	}

	body, err := json.Marshal(event)
	require.NoError(t, err)

	metadata, _ := json.Marshal(map[string]interface{}{
		"agreement_id": event.AgreementID,
		"post_id":      event.PostID,
	})
	metadataStr := string(metadata)

	mock.ExpectExec(`INSERT INTO notifications`).
		WithArgs(event.LenderID, "agreement_created", "New Agreement Request", sqlmock.AnyArg(), metadataStr).
		WillReturnResult(sqlmock.NewResult(1, 1))

	consumer := &Consumer{db: db}

	err = consumer.handleAgreementCreated(body)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleAgreementCancelled(t *testing.T) {
	db, mock := setupTestConsumer(t)

	event := AgreementCancelledEvent{
		AgreementID: 1,
		LenderID:    10,
		BorrowerID:  20,
		PostID:      5,
		Status:      "cancelled",
	}

	body, err := json.Marshal(event)
	require.NoError(t, err)

	metadata, _ := json.Marshal(map[string]interface{}{
		"agreement_id": event.AgreementID,
		"post_id":      event.PostID,
	})
	metadataStr := string(metadata)

	// Two notifications (lender and borrower)
	mock.ExpectExec(`INSERT INTO notifications`).
		WithArgs(event.LenderID, "agreement_cancelled", "Agreement Cancelled", sqlmock.AnyArg(), metadataStr).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO notifications`).
		WithArgs(event.BorrowerID, "agreement_cancelled", "Agreement Cancelled", sqlmock.AnyArg(), metadataStr).
		WillReturnResult(sqlmock.NewResult(2, 1))

	consumer := &Consumer{db: db}

	err = consumer.handleAgreementCancelled(body)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}

