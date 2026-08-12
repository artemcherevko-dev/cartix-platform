package nats

import (
	"context"
	"encoding/json"

	natsclient "nats"
)

const (
	UserRegisteredSubject = "user.registered"
)

type UserRegistered struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
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
