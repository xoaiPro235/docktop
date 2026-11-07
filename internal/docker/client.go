package docker

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/moby/moby/client"
)

type Client struct {
	cli  *client.Client
	host string
}

// Host returns the configured Docker host endpoint, if any.
func (c *Client) Host() string {
	return c.host
}

// NewClient initializes a Docker API client using intelligent multi-tier host resolution.
//
// The optional hosts parameter allows callers to pass an explicit endpoint (e.g. from CLI flag
// -H/--host or config file). If multiple hosts are provided, only the first non-empty one is used.
//
// Host resolution precedence:
//  1. Explicit host argument passed to NewClient (if non-empty)
//  2. DOCKER_HOST environment variable (if set and non-empty)
//  3. Active Docker context from ~/.docker/config.json or $DOCKER_CONTEXT
//  4. Probed local daemon sockets (macOS user sockets, Windows WSL2 named pipe, Linux rootless)
//  5. Standard Moby default socket (/var/run/docker.sock)
//
// Upon successful client initialization, NewClient immediately executes a Ping with a 5-second
// timeout to negotiate the API version and ensure daemon reachability before returning.
func NewClient(hosts ...string) (*Client, error) {
	opts := []client.Opt{client.FromEnv}

	var targetHost string
	if len(hosts) > 0 {
		targetHost = strings.TrimSpace(hosts[0])
	}
	if targetHost == "" {
		if envHost := strings.TrimSpace(os.Getenv("DOCKER_HOST")); envHost != "" {
			targetHost = envHost
		} else {
			targetHost = resolveDefaultHost()
		}
	}
	if targetHost != "" {
		opts = append(opts, client.WithHost(targetHost))
	}

	cli, err := client.New(opts...)
	if err != nil {
		return nil, err
	}

	c := &Client{
		cli:  cli,
		host: targetHost,
	}

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
