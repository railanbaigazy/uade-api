package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL          string
	Port           string
	Env            string
	JWTSecret      string
	RabbitURL      string
	RabbitExchange string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found (ignore this if run within CI/CD or docker-compose)")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set - create a .env file or set it manually")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not set in .env")
	}

	cfg := &Config{
		DBURL:     dbURL,
		Port:      port,
		Env:       env,
		JWTSecret: jwtSecret,
	}

	// RabbitMQ
	cfg.RabbitURL = os.Getenv("RABBITMQ_URL")
	if cfg.RabbitURL == "" {
		cfg.RabbitURL = "amqp://guest:guest@rabbitmq:5672/"
	}

	cfg.RabbitExchange = os.Getenv("RABBITMQ_EXCHANGE")
	if cfg.RabbitExchange == "" {
		cfg.RabbitExchange = "uade.events"
	}

	log.Printf("Loaded config for %s environment", env)

	return cfg
}
