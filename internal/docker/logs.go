package docker

import (
	"bufio"
	"context"
	"io"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

func (c *Client) StreamContainerLogs(ctx context.Context, id string, tail string) <-chan string {
	outCh := make(chan string, 100)

	go func() {
		defer close(outCh)
		inspectRes, err := c.InspectContainer(ctx, id)
		if err != nil {
			return
		}
		isTTY := inspectRes.Config != nil && inspectRes.Config.Tty
		resp, err := c.cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Timestamps: false,
			Tail:       tail,
		})
		if err != nil {
			return
		}
		defer resp.Close()

		if isTTY {
			scanLogs(ctx, resp, outCh)
			return
		}

		pr, pw := io.Pipe()
		defer pr.Close()
		go func() {
			_, err := stdcopy.StdCopy(pw, pw, resp)
			_ = pw.CloseWithError(err)
		}()
		scanLogs(ctx, pr, outCh)
	}()
	return outCh
}

func scanLogs(ctx context.Context, r io.Reader, out chan<- string) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case out <- scanner.Text():
		}
	}
	_ = scanner.Err()
}
