package docker

import (
	"context"
	"strings"
	"time"

	"github.com/moby/moby/client"
)

type Client struct {
	cli *client.Client
}

// NewClient initializes a Docker API client.
// An optional host endpoint can be provided (e.g. "unix:///var/run/docker.sock", "ssh://user@remote", "tcp://ip:2376").
// If no host is provided (or empty), client.FromEnv automatically inspects the DOCKER_HOST environment variable
// or falls back to the default OS Docker socket.
func NewClient(hosts ...string) (*Client, error) {
	opts := []client.Opt{client.FromEnv}

	var targetHost string
	if len(hosts) > 0 {
		targetHost = strings.TrimSpace(hosts[0])
	}
	if targetHost != "" {
		opts = append(opts, client.WithHost(targetHost))
	}

	cli, err := client.New(opts...)
	if err != nil {
		return nil, err
	}

	c := &Client{cli: cli}

	// Negotiate API version immediately upon connection.
	// 5-second timeout handles potential network/SSH latency to remote hosts.
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.cli.Ping(ctx, client.PingOptions{
		NegotiateAPIVersion: true,
	})
	return err
}
