package app

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("PORT", "8080")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("DATABASE_URL", "postgres://user:password@localhost:5430/uade?sslmode=disable")
	_ = os.Setenv("AMQP_URL", "amqp://guest:guest@localhost:5672/")

	os.Exit(m.Run())
}
