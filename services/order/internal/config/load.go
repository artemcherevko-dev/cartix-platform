package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DSNOrders   string
	GRPCPort    string
	NATSUrl     string
	CatalogURL  string
	PaymentsURL string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("warning: .env not loaded")
	}

	dsn, err := loadEnv("DSN_ORDERS")
	if err != nil {
		log.Fatal(err)
	}
	port, err := loadEnv("GRPC_ORDER_PORT")
	if err != nil {
		log.Fatal(err)
	}
	natsURL, err := loadEnv("NATS_URL")
	if err != nil {
		log.Fatal(err)
	}
	catalogURL, err := loadEnv("CATALOG_SERVICE_URL")
	if err != nil {
		log.Fatal(err)
	}
	paymentsURL, err := loadEnv("PAYMENTS_SERVICE_URL")
	if err != nil {
		log.Fatal(err)
	}

	return &Config{
		DSNOrders:   dsn,
		GRPCPort:    port,
		NATSUrl:     natsURL,
		CatalogURL:  catalogURL,
		PaymentsURL: paymentsURL,
	}
}

func loadEnv(name string) (value string, err error) {
	if val, ok := os.LookupEnv(name); ok {
		return val, nil
	}
	return "", errors.New("environment variable " + name + " not set")
}
