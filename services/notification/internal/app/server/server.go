package app

import (
	"context"
	"log"
	natsclient "nats"
	"notification/internal/app/config"
	nats2 "notification/internal/app/nats"
	"os"
	"os/signal"
	"syscall"

	"github.com/wneessen/go-mail"
)

func Run(cfg *config.Config) error {
	smtpClient, err := mail.NewClient(cfg.SMTPUrl, mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(cfg.SMTPUser), mail.WithPassword(cfg.SMTPPass))
	if err != nil {
		return err
	}
	defer smtpClient.Close()

	log.Println("[AUTH] Connecting NATS...")
	natsClient, err := natsclient.New(cfg.NATSUrl)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	mailer := nats2.NewMailer(smtpClient)
	worker := nats2.NewWorker(consumer, mailer)

	if err := worker.Run(ctx, cfg.FromMail); err != nil {
		log.Printf("[PROFILE] Worker stopped: %v", err)
	}

	return nil
}
