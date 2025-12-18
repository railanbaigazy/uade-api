package models

import "time"

type NotificationType string

const (
	NotificationTypeAgreementCreated   NotificationType = "agreement_created"
	NotificationTypeAgreementAccepted  NotificationType = "agreement_accepted"
	NotificationTypeAgreementCancelled NotificationType = "agreement_cancelled"
	NotificationTypePaymentReminder    NotificationType = "payment_reminder"
	NotificationTypePaymentOverdue     NotificationType = "payment_overdue"
	NotificationTypeAgreementCompleted NotificationType = "agreement_completed"
)

type NotificationStatus string

const (
	NotificationStatusUnread   NotificationStatus = "unread"
	NotificationStatusRead     NotificationStatus = "read"
	NotificationStatusArchived NotificationStatus = "archived"
)

type Notification struct {
	ID          int64              `db:"id" json:"id"`
	UserID      int64              `db:"user_id" json:"user_id"`
	Type        NotificationType   `db:"type" json:"type"`
	Title       string             `db:"title" json:"title"`
	Message     string             `db:"message" json:"message"`
	Status      NotificationStatus `db:"status" json:"status"`
	AgreementID *int               `db:"agreement_id" json:"agreement_id,omitempty"`
	Metadata    *string            `db:"metadata" json:"metadata,omitempty"`
	CreatedAt   time.Time          `db:"created_at" json:"created_at"`
	ReadAt      *time.Time         `db:"read_at" json:"read_at,omitempty"`
}
