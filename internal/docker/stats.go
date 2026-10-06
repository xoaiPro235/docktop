package docker

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type ContainerMetrics struct {
	CPUPercent float64
	MemUsageMB float64
	MemLimitMB float64
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

	m.MemUsageMB = usage / (1024 * 1024)
	m.MemLimitMB = float64(v.MemoryStats.Limit) / (1024 * 1024)

	return m
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
		defer resp.Body.Close()

		decoder := jsontext.NewDecoder(resp.Body)
		for {
			var stats container.StatsResponse
			if err := json.UnmarshalDecode(decoder, &stats); err != nil {
				return
			}

			metrics := CalculateStats(&stats)
			select {
			case <-ctx.Done():
				return
			case outCh <- metrics:
			}
		}
	}()
	return outCh
}
