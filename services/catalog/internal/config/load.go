package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DSNCatalog string
	GRPCPort   string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("warning: .env not loaded")
	}

	dsn, err := loadEnv("DSN_CATALOG")
	if err != nil {
		log.Fatal(err)
	}
	port, err := loadEnv("GRPC_CATALOG_PORT")
	if err != nil {
		log.Fatal(err)
	}

	return &Config{
		DSNCatalog: dsn,
		GRPCPort:   port,
	}
}

func loadEnv(name string) (value string, err error) {
	if val, ok := os.LookupEnv(name); ok {
		return val, nil
	}
	return "", errors.New("environment variable " + name + " not set")
}
