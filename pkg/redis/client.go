package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	RDB *redis.Client
}

func New(url string) (*Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opts)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &Client{
		RDB: rdb,
	}, nil
}

func (c *Client) Close() error {
	return c.RDB.Close()
}
