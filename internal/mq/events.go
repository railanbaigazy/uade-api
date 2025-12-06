package mq

import "time"

// AgreementAcceptedEvent represents the event when an agreement is accepted
type AgreementAcceptedEvent struct {
	AgreementID int64      `json:"agreement_id"`
	LenderID    int64      `json:"lender_id"`
	BorrowerID  int64      `json:"borrower_id"`
	PostID      int        `json:"post_id"`
	Status      string     `json:"status"`
	AcceptedAt  *time.Time `json:"accepted_at"`
	StartDate   *time.Time `json:"start_date"`
	DueDate     time.Time  `json:"due_date"`
}

// PaymentReminderEvent represents a payment reminder notification
type PaymentReminderEvent struct {
	AgreementID int64     `json:"agreement_id"`
	UserID      int64     `json:"user_id"`
	DueDate     time.Time `json:"due_date"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
}

// OverdueAlertEvent represents an overdue payment alert
type OverdueAlertEvent struct {
	AgreementID int64     `json:"agreement_id"`
	UserID      int64     `json:"user_id"`
	DueDate     time.Time `json:"due_date"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	DaysOverdue int       `json:"days_overdue"`
}

// AgreementCreatedEvent represents the event when an agreement is created
type AgreementCreatedEvent struct {
	AgreementID      int64     `json:"agreement_id"`
	LenderID         int64     `json:"lender_id"`
	BorrowerID       int64     `json:"borrower_id"`
	PostID           int       `json:"post_id"`
	PrincipalAmount  float64   `json:"principal_amount"`
	InterestRate     float64   `json:"interest_rate"`
	TotalAmount      float64   `json:"total_amount"`
	Currency         string    `json:"currency"`
	DueDate          time.Time `json:"due_date"`
	PaymentFrequency string    `json:"payment_frequency"`
	NumberOfPayments int       `json:"number_of_payments"`
	Status           string    `json:"status"`
}

// AgreementCancelledEvent represents the event when an agreement is cancelled
type AgreementCancelledEvent struct {
	AgreementID int64  `json:"agreement_id"`
	LenderID    int64  `json:"lender_id"`
	BorrowerID  int64  `json:"borrower_id"`
	PostID      int    `json:"post_id"`
	Status      string `json:"status"`
}
