package docker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

var ErrMustStopContainer = errors.New("container must be stopped before removal")

type ContainerInfo struct {
	ID     string
	Name   string
	Image  string
	Status string
	State  container.ContainerState
	Ports  string

	// Doker Compose labels
	IsCompose       bool
	ProjectName     string
	Service         string
	ContainerNumber int
	OneOff          bool

	Details *container.InspectResponse
}

type RemoveContainerOpts struct {
	RemoveVolumes bool
	Force         bool
}

type statusCoder interface {
	StatusCode() int
}

func isConflictError(err error) bool {
	if err == nil {
		return false
	}

	var sc statusCoder
	if errors.As(err, &sc) && sc.StatusCode() == 409 {
		return true
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "container is running") || strings.Contains(errMsg, "conflict")
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func (c *ContainerInfo) ShortID() string {
	return shortID(c.ID)
}

func (c *ContainerInfo) HasDetailsLoaded() bool {
	return c.Details != nil
}

// Docker container cmd

// ListContainers returns a list of containers with basic information.
func (c *Client) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	result, err := c.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("Can not list containers: %w", err)
	}
	var containers []ContainerInfo
	for _, ctr := range result.Items {
		name := shortID(ctr.ID)
		if len(ctr.Names) > 0 {
			name = strings.TrimPrefix(ctr.Names[0], "/")
		}

		var ports []string
		for _, p := range ctr.Ports {
			if p.PublicPort > 0 {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", p.PublicPort, p.PrivatePort, p.Type))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
			}
		}

		info := ContainerInfo{
			ID:     ctr.ID,
			Name:   name,
			Image:  ctr.Image,
			Status: ctr.Status,
			State:  ctr.State,
			Ports:  strings.Join(ports, ", "),
		}

		if project, ok := ctr.Labels["com.docker.compose.project"]; ok {
			info.IsCompose = true
			info.ProjectName = project
			info.Service = ctr.Labels["com.docker.compose.service"]
			info.OneOff = ctr.Labels["com.docker.compose.oneoff"] == "True"

			if numString, ok := ctr.Labels["com.docker.compose.container-number"]; ok {
				if num, err := strconv.Atoi(numString); err == nil {
					info.ContainerNumber = num
				}
			}
		}
		containers = append(containers, info)
	}
	return containers, nil
}

func (c *Client) InspectContainer(ctx context.Context, containerID string) (*container.InspectResponse, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("Can not inspect container %s: %w", containerID, err)
	}
	return &inspect.Container, nil
}

func (c *Client) ContainerTop(ctx context.Context, containerID string) (*container.TopResponse, error) {
	top, err := c.cli.ContainerTop(ctx, containerID, client.ContainerTopOptions{})
	if err != nil {
		return nil, fmt.Errorf("Can not get top info for container %s: %w", containerID, err)
	}
	return &container.TopResponse{Processes: top.Processes, Titles: top.Titles}, nil
}

func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	_, err := c.cli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	return err
}

func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	_, err := c.cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{})
	return err
}

func (c *Client) RestartContainer(ctx context.Context, containerID string) error {
	_, err := c.cli.ContainerRestart(ctx, containerID, client.ContainerRestartOptions{})
	return err
}

func (c *Client) PauseContainer(ctx context.Context, containerID string) error {
	_, err := c.cli.ContainerPause(ctx, containerID, client.ContainerPauseOptions{})
	return err
}

func (c *Client) UnpauseContainer(ctx context.Context, containerID string) error {
	_, err := c.cli.ContainerUnpause(ctx, containerID, client.ContainerUnpauseOptions{})
	return err
}

func (c *Client) RemoveContainer(ctx context.Context, id string, opts RemoveContainerOpts) error {
	_, err := c.cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{
		Force:         opts.Force,
		RemoveVolumes: opts.RemoveVolumes,
	})
	if err != nil {
		if isConflictError(err) {
			return ErrMustStopContainer
		}
		return err
	}
	return nil
}

func (c *Client) PruneContainers(ctx context.Context) (container.PruneReport, error) {
	report, err := c.cli.ContainerPrune(ctx, client.ContainerPruneOptions{})
	if err != nil {
		return container.PruneReport{}, err
	}
	return report.Report, nil
}

func (c *Client) ExecShellCmd(containerID string) *exec.Cmd {
	return exec.Command("docker", "exec", "-it", containerID, "sh", "-c", "bash || sh")
}
