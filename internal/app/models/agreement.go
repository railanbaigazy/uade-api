package models

import "time"

type Agreement struct {
	ID               int        `json:"id" db:"id"`
	LenderID         int64      `json:"lender_id" db:"lender_id"`
	BorrowerID       int64      `json:"borrower_id" db:"borrower_id"`
	PostID           int        `json:"post_id" db:"post_id"`
	PrincipalAmount  float64    `json:"principal_amount" db:"principal_amount"`
	InterestRate     float64    `json:"interest_rate" db:"interest_rate"`
	TotalAmount      float64    `json:"total_amount" db:"total_amount"`
	Currency         string     `json:"currency" db:"currency"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty" db:"accepted_at"`
	DisbursedAt      *time.Time `json:"disbursed_at,omitempty" db:"disbursed_at"`
	StartDate        *time.Time `json:"start_date,omitempty" db:"start_date"`
	DueDate          time.Time  `json:"due_date" db:"due_date"`
	CompletedAt      *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	PaymentFrequency string     `json:"payment_frequency" db:"payment_frequency"`
	NumberOfPayments int        `json:"number_of_payments" db:"number_of_payments"`
	Status           string     `json:"status" db:"status"`
	ContractURL      *string    `json:"contract_url,omitempty" db:"contract_url"`
	ContractHash     *string    `json:"contract_hash,omitempty" db:"contract_hash"`
}
