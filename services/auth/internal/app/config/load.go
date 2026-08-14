package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DSNAuth      string
	JWTSecret    string
	GRPCPortAuth string
	NATSUrl      string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	dsnAuth, err := loadEnv("DSN_AUTH")
	if err != nil {
		log.Fatal(err)
	}
	jwt, err := loadEnv("JWT_SECRET")
	if err != nil {
		log.Fatal(err)
	}
	grpcAuth, err := loadEnv("GRPC_AUTH_PORT")
	if err != nil {
		log.Fatal(err)
	}
	natsURL, err := loadEnv("NATS_URL")
	if err != nil {
		log.Fatal(err)
	}

	return &Config{
		DSNAuth:      dsnAuth,
		JWTSecret:    jwt,
		GRPCPortAuth: grpcAuth,
		NATSUrl:      natsURL,
	}
}

func loadEnv(name string) (value string, err error) {
	if val, ok := os.LookupEnv(name); ok {
		return val, nil
	}
	return "", errors.New("environment variable " + name + " not set")
}
