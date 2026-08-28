package nats

import (
	"context"

	natsclient "nats"

	"github.com/nats-io/nats.go/jetstream"
)

// ensureStreams declares the PAYMENTS stream this service consumes from.
// Stream creation is idempotent and mirrors what the payments service
// declares on its side, so startup order does not matter.
func ensureStreams(ctx context.Context, client *natsclient.Client) error {
	_, err := client.CreateOrUpdateStream(
		ctx,
		jetstream.StreamConfig{
			Name:      "PAYMENTS",
			Subjects:  []string{"payment.>"},
			Retention: jetstream.LimitsPolicy,
		},
	)

	return err
}

func InitConsumer(ctx context.Context, client *natsclient.Client) (jetstream.Consumer, error) {
	if err := ensureStreams(ctx, client); err != nil {
		return nil, err
	}

	return client.JS.CreateOrUpdateConsumer(
		ctx,
		"PAYMENTS",
		jetstream.ConsumerConfig{
			Durable:       "order-paid",
			FilterSubject: "payment.succeeded",
			AckPolicy:     jetstream.AckExplicitPolicy,
		},
	)
}
