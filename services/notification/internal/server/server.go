package server

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	natsclient "nats"
	"notification/internal/config"
	"notification/internal/nats"
	"notification/internal/orders"
	notificationstore "notification/internal/store"
	redisclient "redis"
	telemetry "shared/metrics"

	"github.com/wneessen/go-mail"
)

const shutdownTimeout = 10 * time.Second

func Run(cfg *config.Config) error {
	opts := []mail.Option{
		mail.WithTLSPolicy(mail.TLSOpportunistic),
	}
	if port, err := strconv.Atoi(cfg.SMTPPort); err == nil {
		opts = append(opts, mail.WithPort(port))
	}
	if cfg.SMTPUser != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(cfg.SMTPUser),
			mail.WithPassword(cfg.SMTPPass),
		)
	}

	smtpClient, err := mail.NewClient(cfg.SMTPUrl, opts...)
	if err != nil {
		return err
	}
	defer smtpClient.Close()

	log.Println("[NOTIFICATION] Connecting Redis...")
	redisClient, err := redisclient.New(cfg.RedisUrl)
	if err != nil {
		return err
	}
	defer redisClient.Close()

	log.Println("[NOTIFICATION] Connecting NATS...")
	natsClient, err := natsclient.New(cfg.NATSUrl)
	if err != nil {
		return err
	}
	defer natsClient.Close()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()
	obs, err := telemetry.New(ctx, telemetry.ConfigFromEnv("notification-service"))
	if err != nil {
		return err
	}
	if err := obs.Start(); err != nil {
		return err
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()
		if err := obs.Shutdown(shutdownCtx); err != nil {
			log.Printf("[NOTIFICATION] telemetry shutdown: %v", err)
		}
	}()

	consumer, err := nats.InitConsumer(ctx, natsClient)
	if err != nil {
		return err
	}

	receiptConsumer, err := nats.InitReceiptConsumer(ctx, natsClient)
	if err != nil {
		return err
	}

	statusConsumer, err := nats.InitStatusConsumer(ctx, natsClient)
	if err != nil {
		return err
	}

	mailer := nats.NewMailer(smtpClient, cfg.AppURL, cfg.TemplatesDir)
	dedupe := notificationstore.NewDedupeStore(redisClient, cfg.ReceiptDedupeTTL)

	var ordersClient nats.ReceiptOrders
	if cfg.OrderServiceUrl != "" {
		log.Println("[NOTIFICATION] Connecting Order Service for receipt items...")
		client, err := orders.New(cfg.OrderServiceUrl)
		if err != nil {
			return err
		}
		ordersClient = client
	}

	welcomeWorker := nats.NewWorker(consumer, nats.WelcomeHandler(mailer, cfg.FromMail))
	receiptWorker := nats.NewWorker(receiptConsumer, nats.ReceiptHandler(mailer, dedupe, cfg.FromMail, ordersClient))
	statusWorker := nats.NewWorker(statusConsumer, nats.StatusUpdatedHandler(mailer, cfg.FromMail))

	welcomeDone := make(chan error, 1)
	go func() {
		welcomeDone <- welcomeWorker.Run(ctx)
	}()

	receiptDone := make(chan error, 1)
	go func() {
		receiptDone <- receiptWorker.Run(ctx)
	}()

	statusDone := make(chan error, 1)
	go func() {
		statusDone <- statusWorker.Run(ctx)
	}()

	select {
	case err := <-welcomeDone:
		if err != nil && ctx.Err() == nil {
			return err
		}
	case err := <-receiptDone:
		if err != nil && ctx.Err() == nil {
			return err
		}
	case err := <-statusDone:
		if err != nil && ctx.Err() == nil {
			return err
		}
	case <-ctx.Done():
	}

	log.Println("[NOTIFICATION] Shutting down...")

	for name, done := range map[string]<-chan error{
		"welcome worker": welcomeDone,
		"receipt worker": receiptDone,
		"status worker":  statusDone,
	} {
		select {
		case err := <-done:
			if err != nil && ctx.Err() == nil {
				log.Printf("[NOTIFICATION] %s stopped with error: %v", name, err)
			} else {
				log.Printf("[NOTIFICATION] %s stopped gracefully", name)
			}
		case <-time.After(shutdownTimeout):
			log.Printf("[NOTIFICATION] Waiting for %s timed out", name)
		}
	}

	return nil
}
