package models

import "time"

type Notification struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Type      string    `db:"type" json:"type"`
	Title     string    `db:"title" json:"title"`
	Message   string    `db:"message" json:"message"`
	Read      bool      `db:"read" json:"read"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	ReadAt    *time.Time `db:"read_at" json:"read_at,omitempty"`
	Metadata  *string   `db:"metadata" json:"metadata,omitempty"` // JSON string for additional data
}


