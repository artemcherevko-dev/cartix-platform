package nats

import (
	"context"
	"encoding/json"
	natsclient "nats"
	"time"
)

const (
	UserRegisteredSubject = "user.registered"
)

type UserRegistered struct {
	UserID     string `json:"user_id"`
	FirstName  string `json:"first_name"`
	MiddleName string `json:"middle_name"`
	LastName   string `json:"last_name"`
	Email      string `json:"email"`

	DateOfBirth time.Time `json:"date_of_birth"`
}

func PublishUserRegistered(
	ctx context.Context,
	client *natsclient.Client,
	event UserRegistered,
) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return client.Publish(
		ctx,
		UserRegisteredSubject,
		data,
	)
}
