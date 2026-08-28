package config

import (
	"errors"
	"log"
	"os"
)

type Config struct {
	DSN      string
	GRPCPort string
	NATSUrl  string
}

func LoadConfig() *Config {
	dsn, err := loadEnv("DSN_PROFILE")
	if err != nil {
		log.Fatal("DSN_PROFILE env var not set")
	}
	grpcPort, err := loadEnv("GRPC_PROFILE_PORT")
	if err != nil {
		log.Fatal("GRPC_PORT_PROFILE env var not set")
	}
	natsUrl, err := loadEnv("NATS_URL")
	if err != nil {
		log.Fatal("NATS_URL env var not set")
	}

	return &Config{
		DSN:      dsn,
		GRPCPort: grpcPort,
		NATSUrl:  natsUrl,
	}
}

func loadEnv(name string) (value string, err error) {
	if val, ok := os.LookupEnv(name); ok {
		return val, nil
	}
	return "", errors.New("environment variable " + name + " not set")
}
