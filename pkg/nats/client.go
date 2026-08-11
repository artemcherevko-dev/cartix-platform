package nats

import (
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Client struct {
	Conn *nats.Conn
	JS   jetstream.JetStream
}

func New(url string) (*Client, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Client{
		Conn: conn,
		JS:   js,
	}, nil
}

func (c *Client) Close() {
	c.Conn.Close()
}
