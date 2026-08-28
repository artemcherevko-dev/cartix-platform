package nats

import (
	"context"

	natsclient "nats"

	"github.com/nats-io/nats.go/jetstream"
)

func InitStreams(
	ctx context.Context,
	client *natsclient.Client,
) error {
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
