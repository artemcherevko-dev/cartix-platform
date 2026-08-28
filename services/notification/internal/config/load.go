package config

import (
	"errors"
	"log"
	"os"
	"time"
)

type Config struct {
	NATSUrl  string
	SMTPUrl  string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	FromMail string

	AppURL   string
	RedisUrl string

	OrderServiceUrl string

	TemplatesDir string

	ReceiptDedupeTTL time.Duration
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
	// Auth is optional: local relays like Mailpit usually run without it.
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpPort, err := loadEnv("SMTP_PORT")
	if err != nil {
		log.Fatal("SMTP_PORT env var not set")
	}
	fromMail, err := loadEnv("FROM_MAIL")
	if err != nil {
		log.Fatal("FROM_MAIL env var not set")
	}
	redisUrl, err := loadEnv("REDIS_URL")
	if err != nil {
		log.Fatal("REDIS_URL env var not set")
	}

	appURL := "http://localhost:3000"
	if val, ok := os.LookupEnv("APP_URL"); ok && val != "" {
		appURL = val
	}

	templatesDir := "./templates"
	if val, ok := os.LookupEnv("TEMPLATE_DIR"); ok && val != "" {
		templatesDir = val
	}

	receiptDedupeTTL := 7 * 24 * time.Hour
	if raw, ok := os.LookupEnv("RECEIPT_DEDUPE_TTL"); ok {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			log.Fatal("invalid RECEIPT_DEDUPE_TTL: " + err.Error())
		}
		receiptDedupeTTL = parsed
	}

	return &Config{
		NATSUrl:  natsUrl,
		SMTPUrl:  smtpUrl,
		SMTPPort: smtpPort,
		SMTPUser: smtpUser,
		SMTPPass: smtpPass,
		FromMail: fromMail,

		AppURL:   appURL,
		RedisUrl: redisUrl,

		OrderServiceUrl: os.Getenv("ORDER_SERVICE_URL"),

		TemplatesDir: templatesDir,

		ReceiptDedupeTTL: receiptDedupeTTL,
	}
}

func loadEnv(name string) (value string, err error) {
	if val, ok := os.LookupEnv(name); ok {
		return val, nil
	}
	return "", errors.New("environment variable " + name + " not set")
}
