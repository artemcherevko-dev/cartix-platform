package nats

import (
	"context"

	natsclient "nats"

	"github.com/nats-io/nats.go/jetstream"
)

// ensureStreams declares every stream this service consumes from. Stream
// creation is idempotent, so declaring them here keeps the worker resilient
// to startup order (publishers declare the same streams on their side).
func ensureStreams(ctx context.Context, client *natsclient.Client) error {
	for _, stream := range []jetstream.StreamConfig{
		{
			Name:      "USERS",
			Subjects:  []string{"user.>"},
			Retention: jetstream.LimitsPolicy,
		},
		{
			Name:      "PAYMENTS",
			Subjects:  []string{"payment.>"},
			Retention: jetstream.LimitsPolicy,
		},
		{
			Name:      "ORDERS",
			Subjects:  []string{"order.>"},
			Retention: jetstream.LimitsPolicy,
		},
	} {
		if _, err := client.CreateOrUpdateStream(ctx, stream); err != nil {
			return err
		}
	}

	return nil
}

func InitConsumer(ctx context.Context, client *natsclient.Client) (jetstream.Consumer, error) {
	if err := ensureStreams(ctx, client); err != nil {
		return nil, err
	}

	return client.JS.CreateOrUpdateConsumer(
		ctx,
		"USERS",
		jetstream.ConsumerConfig{
			Durable:       "notification-welcome",
			FilterSubject: "user.registered",
			AckPolicy:     jetstream.AckExplicitPolicy,
		},
	)
}

func InitReceiptConsumer(ctx context.Context, client *natsclient.Client) (jetstream.Consumer, error) {
	if err := ensureStreams(ctx, client); err != nil {
		return nil, err
	}

	return client.JS.CreateOrUpdateConsumer(
		ctx,
		"PAYMENTS",
		jetstream.ConsumerConfig{
			Durable:       "receipt",
			FilterSubject: "payment.succeeded",
			AckPolicy:     jetstream.AckExplicitPolicy,
		},
	)
}

func InitStatusConsumer(ctx context.Context, client *natsclient.Client) (jetstream.Consumer, error) {
	if err := ensureStreams(ctx, client); err != nil {
		return nil, err
	}

	return client.JS.CreateOrUpdateConsumer(
		ctx,
		"ORDERS",
		jetstream.ConsumerConfig{
			Durable:       "notification-status",
			FilterSubject: "order.status_updated",
			AckPolicy:     jetstream.AckExplicitPolicy,
		},
	)
}
