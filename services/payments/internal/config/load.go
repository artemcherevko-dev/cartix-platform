package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DSNPayments string
	NATSUrl     string

	HTTPPort string
	GRPCPort string

	StripeWebhookSecret string
	StripeSecretKey     string
	StripeAPIURL        string

	AppURL string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("warning: .env not loaded")
	}

	dsn, err := loadEnv("DSN_PAYMENTS")
	if err != nil {
		log.Fatal(err)
	}
	natsURL, err := loadEnv("NATS_URL")
	if err != nil {
		log.Fatal(err)
	}
	httpPort, err := loadEnv("PAYMENTS_HTTP_PORT")
	if err != nil {
		log.Fatal(err)
	}
	grpcPort, err := loadEnv("GRPC_PAYMENTS_PORT")
	if err != nil {
		log.Fatal(err)
	}
	webhookSecret, err := loadEnv("STRIPE_WEBHOOK_SECRET")
	if err != nil {
		log.Fatal(err)
	}
	stripeKey, err := loadEnv("STRIPE_SECRET_KEY")
	if err != nil {
		log.Fatal(err)
	}

	appURL := "http://localhost:3000"
	if val, ok := os.LookupEnv("APP_URL"); ok && val != "" {
		appURL = val
	}

	return &Config{
		DSNPayments:         dsn,
		NATSUrl:             natsURL,
		HTTPPort:            httpPort,
		GRPCPort:            grpcPort,
		StripeWebhookSecret: webhookSecret,
		StripeSecretKey:     stripeKey,
		StripeAPIURL:        os.Getenv("STRIPE_API_URL"),
		AppURL:              appURL,
	}
}

func loadEnv(name string) (value string, err error) {
	if val, ok := os.LookupEnv(name); ok {
		return val, nil
	}
	return "", errors.New("environment variable " + name + " not set")
}
