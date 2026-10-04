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
	if err != nil {
		return nil, err
	}
	c := &Client{cli: cli}

	if err := c.Ping(); err != nil {
		_ = cli.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}

func (c *Client) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := c.cli.Ping(ctx, client.PingOptions{
		NegotiateAPIVersion: true,
	})
	return err
}
