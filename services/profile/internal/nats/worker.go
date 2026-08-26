package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
)

type ProfileService interface {
	CreateProfile(ctx context.Context, userID uuid.UUID, firstName, middleName, lastName string, dateOfBirth time.Time) error
}

type Worker struct {
	consumer jetstream.Consumer
	service  ProfileService
}

func NewWorker(service ProfileService, consumer jetstream.Consumer) Worker {
	return Worker{
		consumer: consumer,
		service:  service,
	}
}

func (w *Worker) handle(ctx context.Context, msg jetstream.Msg) error {
	var event struct {
		UserID     string `json:"user_id"`
		FirstName  string `json:"first_name"`
		MiddleName string `json:"middle_name"`
		LastName   string `json:"last_name"`

		DateOfBirth time.Time `json:"date_of_birth"`
	}

	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		return fmt.Errorf("unmarshal user.created event: %w", err)
	}

	if err := w.service.CreateProfile(ctx, uuid.MustParse(event.UserID), event.FirstName, event.MiddleName, event.LastName, event.DateOfBirth); err != nil {
		return fmt.Errorf("create profile: %w", err)
	}

	return nil
}

func (w *Worker) Run(ctx context.Context) error {
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
			if err := w.handle(ctx, msg); err != nil {
				log.Printf("handle message: %v", err)
				continue
			}

			if err := msg.Ack(); err != nil {
				log.Printf("ack message: %v", err)
			}
		}
	}
}
