package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go/jetstream"
)

type Worker struct {
	consumer jetstream.Consumer
	mailer   *Mailer
}

func NewWorker(consumer jetstream.Consumer, mailer *Mailer) Worker {
	return Worker{
		consumer: consumer,
		mailer:   mailer,
	}
}

func (w *Worker) handle(msg jetstream.Msg, from string) error {
	var event struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	}

	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		return fmt.Errorf("unmarshal user.created event: %w", err)
	}

	if err := w.mailer.SendMail(from, event.Email, "Email confirmation", UserTemplateData{
		FirstName: event.FirstName,
		LastName:  event.LastName,
	}); err != nil {
		return fmt.Errorf("create profile: %w", err)
	}

	return nil
}

func (w *Worker) Run(ctx context.Context, from string) error {
	for {
		msgs, err := w.consumer.Fetch(10)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			log.Printf("fetch messages: %v", err)
			continue
		}

		for msg := range msgs.Messages() {
			if err := w.handle(msg, from); err != nil {
				log.Printf("handle message: %v", err)
				continue
			}

			if err := msg.Ack(); err != nil {
				log.Printf("ack message: %v", err)
			}
		}
	}
}
