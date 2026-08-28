package nats

import (
	"context"
	natsclient "nats"

	"github.com/nats-io/nats.go/jetstream"
)

func InitConsumer(ctx context.Context, client *natsclient.Client) (jetstream.Consumer, error) {
	return client.JS.CreateOrUpdateConsumer(
		ctx,
		"USERS",
		jetstream.ConsumerConfig{
			Durable:       "profile",
			FilterSubject: "user.registered",
			AckPolicy:     jetstream.AckExplicitPolicy,
		},
	)
}
