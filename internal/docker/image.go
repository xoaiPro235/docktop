package docker

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

type ImageInfo struct {
	ID               string
	RepoTags         []string
	Tag              string
	Size             int64
	Created          time.Time
	InUse            bool
	Dangling         bool
	UsedByContainers []string
	Details          *image.InspectResponse
	Layers           []ImageLayer
}

type ImageLayer struct {
	Index   int
	ID      string
	Size    int64
	Command string
	Created time.Time
	Comment string
}

// CleanLayerCommand returns a single-line, Dockerfile-like instruction
// from a history CreatedBy string.
func CleanLayerCommand(cmd string) string {
	return strings.Join(strings.Fields(cleanLayerCommand(cmd)), " ")
}

func cleanLayerCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "<missing>"
	}

	// Dockerfile directives: /bin/sh -c #(nop)  <INSTRUCTION>
	const nopPrefix = "/bin/sh -c #(nop)"
	if strings.HasPrefix(cmd, nopPrefix) {
		return strings.TrimSpace(strings.TrimPrefix(cmd, nopPrefix))
	}

	// Shell execution: /bin/sh -c <COMMAND> -> RUN <COMMAND>
	const shPrefix = "/bin/sh -c"
	if strings.HasPrefix(cmd, shPrefix) {
		cmd = strings.TrimSpace(strings.TrimPrefix(cmd, shPrefix))
		if !strings.HasPrefix(cmd, "RUN ") {
			cmd = "RUN " + cmd
		}
		return cmd
	}

	return cmd
}

// FormatTimeAgo formats a time into a relative "X ago" string.
func FormatTimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	return units.HumanDuration(d) + " ago"
}

func cleanImageID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 10 {
		return id[:10]
	}
	return id
}

func displayTag(raw image.Summary) (string, bool) {
	for _, t := range raw.RepoTags {
		if t != "<none>:<none>" {
			return t, false
		}
	}
	for _, d := range raw.RepoDigests {
		if !strings.HasPrefix(d, "<none>") {
			repo, _, _ := strings.Cut(d, "@")
			return repo + ":<digest>", false
		}
	}
	return "<none>:<none>", true
}

// ListImages retrieves images and annotates them with container usage and status.
func (c *Client) ListImages(ctx context.Context) ([]ImageInfo, error) {
	imgResult, err := c.cli.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	// Cross-reference containers to know which images are in use.
	ctrResult, err := c.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	usedByID := make(map[string][]string)
	for _, ctr := range ctrResult.Items {
		ctrName := shortID(ctr.ID)
		if len(ctr.Names) > 0 {
			ctrName = strings.TrimPrefix(ctr.Names[0], "/")
		}

		imgID := strings.TrimPrefix(ctr.ImageID, "sha256:")
		if imgID != "" {
			usedByID[imgID] = append(usedByID[imgID], ctrName)
		}
	}

	for _, names := range usedByID {
		slices.Sort(names)
	}

	images := make([]ImageInfo, 0, len(imgResult.Items))
	for _, raw := range imgResult.Items {
		createdTime := time.Unix(raw.Created, 0)
		primaryTag, isDangling := displayTag(raw)

		containersUsing := usedByID[strings.TrimPrefix(raw.ID, "sha256:")]

		images = append(images, ImageInfo{
			ID:               raw.ID,
			RepoTags:         raw.RepoTags,
			Tag:              primaryTag,
			Size:             raw.Size,
			Created:          createdTime,
			InUse:            len(containersUsing) > 0,
			Dangling:         isDangling,
			UsedByContainers: containersUsing,
		})
	}

	// Sort: IN-USE > UNUSED > DANGLING, then newest first, then tag.
	slices.SortStableFunc(images, func(a, b ImageInfo) int {
		if c := cmp.Compare(imageStatusScore(a), imageStatusScore(b)); c != 0 {
			return c
		}
		if !a.Created.Equal(b.Created) {
			if a.Created.After(b.Created) {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.Tag, b.Tag)
	})

	return images, nil
}

func imageStatusScore(img ImageInfo) int {
	if img.InUse {
		return 1
	}
	if img.Dangling {
		return 3
	}
	return 2
}

// InspectImage fetches full inspection metadata for an image.
func (c *Client) InspectImage(ctx context.Context, imageID string) (*image.InspectResponse, error) {
	inspect, err := c.cli.ImageInspect(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image %s: %w", imageID, err)
	}
	return &inspect.InspectResponse, nil
}

// GetImageHistory fetches layers history for an image.
func (c *Client) GetImageHistory(ctx context.Context, imageID string) ([]ImageLayer, error) {
	hist, err := c.cli.ImageHistory(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for image %s: %w", imageID, err)
	}

	layers := make([]ImageLayer, 0, len(hist.Items))
	for i, item := range hist.Items {
		layers = append(layers, ImageLayer{
			Index:   i + 1,
			ID:      cleanImageID(item.ID),
			Size:    item.Size,
			Command: CleanLayerCommand(item.CreatedBy),
			Created: time.Unix(item.Created, 0),
			Comment: item.Comment,
		})
	}
	return layers, nil
}

// RemoveImage deletes an image by ID or tag.
func (c *Client) RemoveImage(ctx context.Context, imageID string, force bool) error {
	_, err := c.cli.ImageRemove(ctx, imageID, client.ImageRemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
	return err
}

// PruneImages removes dangling images.
func (c *Client) PruneImages(ctx context.Context) (image.PruneReport, error) {
	res, err := c.cli.ImagePrune(ctx, client.ImagePruneOptions{})
	if err != nil {
		return image.PruneReport{}, err
	}
	return res.Report, nil
}
