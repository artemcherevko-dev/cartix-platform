package config

import (
	"errors"
	"log"
	"os"
)

type Config struct {
	NATSUrl  string
	SMTPUrl  string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	FromMail string

	AppURL string
}

func LoadConfig() *Config {
	natsUrl, err := loadEnv("NATS_URL")
	if err != nil {
		log.Fatal("NATS_URL env var not set")
	}
	smtpUrl, err := loadEnv("SMTP_URL")
	if err != nil {
		log.Fatal("SMTP_URL env var not set")
	}
	smtpPort, err := loadEnv("SMTP_PORT")
	if err != nil {
		log.Fatal("SMTP_PORT env var not set")
	}
	smtpUser, err := loadEnv("SMTP_USER")
	if err != nil {
		log.Fatal("SMTP_USER env var not set")
	}
	smtpPass, err := loadEnv("SMTP_PASS")
	if err != nil {
		log.Fatal("SMTP_PASS env var not set")
	}
	fromMail, err := loadEnv("FROM_MAIL")
	if err != nil {
		log.Fatal("FROM_MAIL env var not set")
	}
	appURL, err := loadEnv("APP_URL")
	if err != nil {
		log.Fatal("APP_URL env var not set")
	}

	return &Config{
		NATSUrl:  natsUrl,
		SMTPUrl:  smtpUrl,
		SMTPPort: smtpPort,
		SMTPUser: smtpUser,
		SMTPPass: smtpPass,
		FromMail: fromMail,

		AppURL: appURL,
	}
}

func loadEnv(name string) (value string, err error) {
	if val, ok := os.LookupEnv(name); ok {
		return val, nil
	}
	return "", errors.New("environment variable " + name + " not set")
}
