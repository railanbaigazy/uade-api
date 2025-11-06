package models

import "time"

type User struct {
	ID           int64     `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         string    `db:"role" json:"role"`
	State        string    `db:"state" json:"state"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
