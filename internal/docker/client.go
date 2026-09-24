package docker

import (
	"context"
	"time"

	"github.com/moby/moby/client"
)

type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	cli, err := client.New(client.FromEnv)
	return &Client{cli: cli}, err
}

func (c *Client) Close() error {
	return c.cli.Close()
}

func (c *Client) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := c.cli.Ping(ctx, client.PingOptions{
		NegotiateAPIVersion: true,
	})
	return err
}
