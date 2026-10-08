package docker

import (
	"context"
	"os/exec"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
)

// ContainerService defines the narrow interface required by Container components.
type ContainerService interface {
	ListContainers(ctx context.Context) ([]ContainerInfo, error)
	InspectContainer(ctx context.Context, containerID string) (*container.InspectResponse, error)
	ContainerTop(ctx context.Context, containerID string) (*container.TopResponse, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	RestartContainer(ctx context.Context, containerID string) error
	PauseContainer(ctx context.Context, containerID string) error
	UnpauseContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, id string, opts RemoveContainerOpts) error
	PruneContainers(ctx context.Context) (container.PruneReport, error)
	ExecShellCmd(containerID string) *exec.Cmd
	StreamContainerLogs(ctx context.Context, containerID string, tail string) <-chan string
	StreamContainerStats(ctx context.Context, id string) <-chan ContainerMetrics
}

// ImageService defines the narrow interface required by Image components.
type ImageService interface {
	ListImages(ctx context.Context) ([]ImageInfo, error)
	InspectImage(ctx context.Context, id string) (*image.InspectResponse, error)
	GetImageHistory(ctx context.Context, id string) ([]ImageLayer, error)
	RemoveImage(ctx context.Context, id string, force bool) error
	PruneImages(ctx context.Context) (image.PruneReport, error)
}

// VolumeService defines the narrow interface required by Volume components.
type VolumeService interface {
	ListVolumes(ctx context.Context) ([]VolumeInfo, error)
	InspectVolume(ctx context.Context, name string) (*volume.Volume, error)
	RemoveVolume(ctx context.Context, name string, force bool) error
	PruneVolumes(ctx context.Context) (volume.PruneReport, error)
}

// NetworkService defines the narrow interface required by Network components.
type NetworkService interface {
	ListNetworks(ctx context.Context) ([]NetworkInfo, error)
	InspectNetwork(ctx context.Context, id string) (*network.Inspect, error)
	RemoveNetwork(ctx context.Context, id string) error
	PruneNetworks(ctx context.Context) (network.PruneReport, error)
}

// EventService defines daemon lifecycle streaming.
type EventService interface {
	StreamEvents(ctx context.Context) (<-chan events.Message, <-chan error)
}

// Service is an idiomatic composite interface combining all domain sub-services.
// Used by the root AppModel to orchestrate all tabs and engine events.
type Service interface {
	ContainerService
	ImageService
	VolumeService
	NetworkService
	EventService
	Close() error
	Ping() error
}

var (
	_ Service          = (*Client)(nil)
	_ ContainerService = (*Client)(nil)
	_ ImageService     = (*Client)(nil)
	_ VolumeService    = (*Client)(nil)
	_ NetworkService   = (*Client)(nil)
	_ EventService     = (*Client)(nil)
)
