package config

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DSNAuth      string
	JWTSecret    string
	GRPCPortAuth string
	NATSUrl      string
	RedisUrl     string

	VerifyEmailTokenTTL time.Duration
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
	redisURL, err := loadEnv("REDIS_URL")
	if err != nil {
		log.Fatal(err)
	}

	verifyTTL := 15 * time.Minute
	if raw, ok := os.LookupEnv("VERIFY_EMAIL_TOKEN_TTL"); ok {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			log.Fatal("invalid VERIFY_EMAIL_TOKEN_TTL: " + err.Error())
		}
		verifyTTL = parsed
	}

	return &Config{
		DSNAuth:      dsnAuth,
		JWTSecret:    jwt,
		GRPCPortAuth: grpcAuth,
		NATSUrl:      natsURL,
		RedisUrl:     redisURL,

		VerifyEmailTokenTTL: verifyTTL,
	}
}

func loadEnv(name string) (value string, err error) {
	if val, ok := os.LookupEnv(name); ok {
		return val, nil
	}
	return "", errors.New("environment variable " + name + " not set")
}
