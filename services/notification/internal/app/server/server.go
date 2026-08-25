package app

import (
	"context"
	"log"
	natsclient "nats"
	"notification/internal/app/config"
	nats2 "notification/internal/app/nats"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/wneessen/go-mail"
)

const shutdownTimeout = 10 * time.Second

func Run(cfg *config.Config) error {
	opts := []mail.Option{
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(cfg.SMTPUser),
		mail.WithPassword(cfg.SMTPPass),
		mail.WithTLSPolicy(mail.TLSOpportunistic),
	}
	if port, err := strconv.Atoi(cfg.SMTPPort); err == nil {
		opts = append(opts, mail.WithPort(port))
	}

	smtpClient, err := mail.NewClient(cfg.SMTPUrl, opts...)
	if err != nil {
		return err
	}
	defer smtpClient.Close()

	log.Println("[AUTH] Connecting NATS...")
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
	consumer, err := nats2.InitConsumer(ctx, natsClient)
	if err != nil {
		return err
	}

	mailer := nats2.NewMailer(smtpClient, cfg.AppURL)
	worker := nats2.NewWorker(consumer, mailer)

	workerDone := make(chan error, 1)
	go func() {
		workerDone <- worker.Run(ctx, cfg.FromMail)
	}()

	select {
	case err := <-workerDone:
		if err != nil && ctx.Err() == nil {
			return err
		}
	case <-ctx.Done():
	}

	log.Println("[NOTIFICATION] Shutting down...")

	select {
	case err := <-workerDone:
		if err != nil {
			log.Printf("[NOTIFICATION] Worker stopped: %v", err)
		} else {
			log.Println("[NOTIFICATION] Worker stopped gracefully")
		}
	case <-time.After(shutdownTimeout):
		log.Println("[NOTIFICATION] Waiting for worker timed out")
	}

	return nil
}
