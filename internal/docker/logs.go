package docker

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

type ComposeLogOpts struct {
	WorkingDir  string
	ConfigFile  string
	ProjectName string
	Service     string
	Tail        string
}

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
		defer resp.Close() //nolint: errcheck

		if isTTY {
			scanLogs(ctx, resp, outCh)
			return
		}

		pr, pw := io.Pipe()
		defer pr.Close() //nolint:errcheck
		go func() {
			_, err := stdcopy.StdCopy(pw, pw, resp)
			_ = pw.CloseWithError(err)
		}()
		scanLogs(ctx, pr, outCh)
	}()
	return outCh
}

func (c *Client) StreamComposeLogs(ctx context.Context, opts ComposeLogOpts) <-chan string {
	outCh := make(chan string, 100)

	go func() {
		defer close(outCh)

		tail := opts.Tail
		if tail == "" {
			tail = "50"
		}

		args := []string{"compose", "--ansi", "always"}
		if opts.ProjectName != "" {
			args = append(args, "-p", opts.ProjectName)
		}
		if opts.ConfigFile != "" {
			args = append(args, "-f", opts.ConfigFile)
		}
		if opts.WorkingDir != "" {
			args = append(args, "--project-directory", opts.WorkingDir)
		}
		args = append(args, "logs", "-f", "--tail", tail)
		if opts.Service != "" {
			args = append(args, opts.Service)
		}

		cmd := exec.CommandContext(ctx, "docker", args...)
		if opts.WorkingDir != "" {
			cmd.Dir = opts.WorkingDir
		}
		setProcessGroup(cmd)

		cmd.WaitDelay = 200 * time.Millisecond

		pr, pw, err := os.Pipe()
		if err != nil {
			select {
			case <-ctx.Done():
			case outCh <- "Error creating pipe: " + err.Error():
			}
			return
		}
		defer pr.Close() //nolint: errcheck

		cmd.Stdout = pw
		cmd.Stderr = pw

		if err := cmd.Start(); err != nil {
			_ = pw.Close()
			select {
			case <-ctx.Done():
			case outCh <- "Error starting docker compose: " + err.Error():
			}
			return
		}
		// Close parent's copy of pw so only child holds the write end.
		// When child exits or is killed, kernel automatically closes write FD and delivers EOF to pr.
		_ = pw.Close()

		scanLogs(ctx, pr, outCh)

		_ = cmd.Wait()
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
