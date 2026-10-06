package docker

import (
	"context"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

func (c *Client) StreamEvents(ctx context.Context) (<-chan events.Message, <-chan error) {
	filters := make(client.Filters).Add("type",
		string(events.ContainerEventType),
		string(events.ImageEventType),
		string(events.VolumeEventType),
		string(events.NetworkEventType))
	result := c.cli.Events(ctx, client.EventsListOptions{Filters: filters})
	return result.Messages, result.Err
}

// IsContainerLifecycleEvent returns true if the event represents an actual state/lifecycle change of a container.
func IsContainerExecEvent(action events.Action) bool {
	switch action {
	case events.ActionCreate,
		events.ActionStart,
		events.ActionStop,
		events.ActionDie,
		events.ActionKill,
		events.ActionRestart,
		events.ActionPause,
		events.ActionUnPause,
		events.ActionDestroy,
		events.ActionRename,
		events.ActionUpdate,
		events.ActionPrune:
		return true
	default:
		return false

	}
}

// IsImageLifecycleEvent returns true if the event represents an addition, removal, or tag change of an image.
func IsImageLifecycleEvent(action events.Action) bool {
	switch action {
	case events.ActionPull,
		events.ActionDelete,
		events.ActionTag,
		events.ActionUnTag,
		events.ActionImport,
		events.ActionPrune,
		events.ActionSave,
		events.ActionLoad:
		return true
	default:
		return false
	}
}

// IsVolumeLifecycleEvent returns true if the event represents a volume lifecycle modification.
func IsVolumeLifecycleEvent(action events.Action) bool {
	switch action {
	case events.ActionCreate,
		events.ActionDestroy,
		events.ActionMount,
		events.ActionUnmount,
		events.ActionPrune:
		return true
	default:
		return false
	}
}

// IsNetworkLifecycleEvent returns true if the event represents a network lifecycle modification.
func IsNetworkLifecycleEvent(action events.Action) bool {
	switch action {
	case events.ActionCreate,
		events.ActionDestroy,
		events.ActionConnect,
		events.ActionDisconnect,
		events.ActionPrune,
		events.ActionRemove:
		return true
	default:
		return false
	}
}
