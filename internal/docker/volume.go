package docker

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
)

// VolumeContainerMount describes a container currently mounting a volume
type VolumeContainerMount struct {
	ContainerID   string
	ContainerName string
	State         container.ContainerState
	Destination   string
	Mode          string
	RW            bool
}

// VolumeInfo encapsulates metadata and live container usage for a Docker volume
type VolumeInfo struct {
	volume.Volume
	InUse      bool
	RefCnt     int
	Containers []VolumeContainerMount
}

// ListVolumes retrieves all docker volumes and cross-references them with active/stopped containers
func (c *Client) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	volRes, err := c.cli.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	ctrRes, _ := c.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	volumeMounts := make(map[string][]VolumeContainerMount)
	for _, ctr := range ctrRes.Items {
		ctrName := shortID(ctr.ID)
		if len(ctr.Names) > 0 {
			ctrName = strings.TrimPrefix(ctr.Names[0], "/")
		}

		for _, m := range ctr.Mounts {
			if m.Type == mount.TypeVolume && m.Name != "" {
				volumeMounts[m.Name] = append(volumeMounts[m.Name], VolumeContainerMount{
					ContainerID:   ctr.ID,
					ContainerName: ctrName,
					State:         ctr.State,
					Destination:   m.Destination,
					Mode:          m.Mode,
					RW:            m.RW,
				})
			}
		}
	}

	var volumes []VolumeInfo
	for _, v := range volRes.Items {
		mounts := volumeMounts[v.Name]
		slices.SortStableFunc(mounts, func(a, b VolumeContainerMount) int {
			return cmp.Compare(a.ContainerName, b.ContainerName)
		})
		inUse := len(mounts) > 0

		volumes = append(volumes, VolumeInfo{
			Volume:     v,
			InUse:      inUse,
			RefCnt:     len(mounts),
			Containers: mounts,
		})
	}

	// Sort volumes:
	// 1. Status: IN-USE (score 1) > UNUSED/DANGLING (score 2)
	// 2. RefCount descending
	// 3. Name alphabetical
	slices.SortStableFunc(volumes, func(a, b VolumeInfo) int {
		if a.InUse != b.InUse {
			if a.InUse {
				return -1
			}
			return 1
		}
		if c := cmp.Compare(b.RefCnt, a.RefCnt); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})

	return volumes, nil
}

// InspectVolume retrieves deep inspect data for a volume
func (c *Client) InspectVolume(ctx context.Context, name string) (*volume.Volume, error) {
	res, err := c.cli.VolumeInspect(ctx, name, client.VolumeInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume %s: %w", name, err)
	}
	return &res.Volume, nil
}

// RemoveVolume deletes a volume by name
func (c *Client) RemoveVolume(ctx context.Context, name string, force bool) error {
	_, err := c.cli.VolumeRemove(ctx, name, client.VolumeRemoveOptions{Force: force})
	return err
}

// PruneVolumes removes all unused local volumes
func (c *Client) PruneVolumes(ctx context.Context) (volume.PruneReport, error) {
	res, err := c.cli.VolumePrune(ctx, client.VolumePruneOptions{})
	if err != nil {
		return volume.PruneReport{}, err
	}
	return res.Report, nil
}
