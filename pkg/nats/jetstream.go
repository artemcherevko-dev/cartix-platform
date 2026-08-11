package nats

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"
)

func (c *Client) CreateOrUpdateStream(
	ctx context.Context,
	config jetstream.StreamConfig,
) (jetstream.Stream, error) {
	return c.JS.CreateOrUpdateStream(ctx, config)
}

func (c *Client) Publish(
	ctx context.Context,
	subject string,
	data []byte,
) error {
	_, err := c.JS.Publish(ctx, subject, data)
	return err
}
