package docker

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type ContainerMetrics struct {
	CPUPercent  float64
	MemUsage    float64
	MemLimit    float64
	HasMemLimit bool
	NetworkRx   float64
	NetworkTx   float64
	BlockRead   uint64
	BlockWrite  uint64
}

func CalculateStats(v *container.StatsResponse) ContainerMetrics {
	var m ContainerMetrics
	if v == nil {
		return m
	}

	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	onlineCPUs := float64(v.CPUStats.OnlineCPUs)
	if onlineCPUs == 0.0 {
		onlineCPUs = float64(len(v.CPUStats.CPUUsage.PercpuUsage))
	}
	if onlineCPUs == 0.0 {
		onlineCPUs = 1.0
	}

	if systemDelta > 0.0 && cpuDelta > 0.0 && v.PreCPUStats.CPUUsage.TotalUsage > 0 {
		m.CPUPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}

	usage := float64(v.MemoryStats.Usage)
	if cache, ok := v.MemoryStats.Stats["inactive_file"]; ok {
		usage -= float64(cache) // cgroup v2
	} else if cache, ok := v.MemoryStats.Stats["total_inactive_file"]; ok {
		usage -= float64(cache) // cgroup v1
	}

	m.MemUsage = usage
	m.MemLimit = float64(v.MemoryStats.Limit)
	m.NetworkRx, m.NetworkTx = calculateNetwork(v.Networks)
	m.BlockRead, m.BlockWrite = calculateBlockIO(v.BlkioStats)

	return m
}

func calculateNetwork(network map[string]container.NetworkStats) (float64, float64) {
	var rx, tx float64
	for _, v := range network {
		rx += float64(v.RxBytes)
		tx += float64(v.TxBytes)
	}
	return rx, tx
}

func calculateBlockIO(blkio container.BlkioStats) (uint64, uint64) {
	var blkRead, blkWrite uint64
	for _, bioEntry := range blkio.IoServiceBytesRecursive {
		if len(bioEntry.Op) == 0 {
			continue
		}
		switch bioEntry.Op[0] {
		case 'r', 'R':
			blkRead += bioEntry.Value
		case 'w', 'W':
			blkWrite += bioEntry.Value
		}
	}
	return blkRead, blkWrite
}

func (c *Client) StreamContainerStats(ctx context.Context, containerID string) <-chan ContainerMetrics {
	outCh := make(chan ContainerMetrics, 10)

	go func() {
		defer close(outCh)

		resp, err := c.cli.ContainerStats(ctx, containerID, client.ContainerStatsOptions{
			Stream: true,
		})
		if err != nil {
			return
		}
		defer resp.Body.Close() //nolint: errcheck

		var hasMemLimit bool
		var explicitLimit float64
		if insp, err := c.InspectContainer(ctx, containerID); err == nil && insp != nil && insp.HostConfig != nil {
			if insp.HostConfig.Memory > 0 {
				hasMemLimit = true
				explicitLimit = float64(insp.HostConfig.Memory)
			}
		}

		decoder := jsontext.NewDecoder(resp.Body)
		for {
			var stats container.StatsResponse
			if err := json.UnmarshalDecode(decoder, &stats); err != nil {
				return
			}

			metrics := CalculateStats(&stats)
			metrics.HasMemLimit = hasMemLimit
			if hasMemLimit && explicitLimit > 0 {
				metrics.MemLimit = explicitLimit
			}
			select {
			case <-ctx.Done():
				return
			case outCh <- metrics:
			}
		}
	}()
	return outCh
}
