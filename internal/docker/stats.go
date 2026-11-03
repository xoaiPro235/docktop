// Portions of stats calculation logic (CPU, Memory, Disk I/O) are derived from
// docker/cli (https://github.com/docker/cli), licensed under the Apache License, Version 2.0.
// Copyright (c) 2013-2026 Docker, Inc.

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

	isWindows := v.OSType == "windows" || v.NumProcs > 0

	// 1. CPU Percentage
	if isWindows {
		m.CPUPercent = calculateCPUPercentWindows(v)
	} else {
		m.CPUPercent = calculateCPUPercentUnix(v)
	}

	// 2. Memory Usage
	if isWindows {
		m.MemUsage = float64(v.MemoryStats.PrivateWorkingSet)
	} else {
		usage := float64(v.MemoryStats.Usage)
		// cgroup v1: "total_inactive_file"
		if cache, ok := v.MemoryStats.Stats["total_inactive_file"]; ok && cache < v.MemoryStats.Usage {
			usage = float64(v.MemoryStats.Usage - cache)
			// cgroup v2: "inactive_file"
		} else if cache, ok := v.MemoryStats.Stats["inactive_file"]; ok && cache < v.MemoryStats.Usage {
			usage = float64(v.MemoryStats.Usage - cache)
		}
		m.MemUsage = usage
	}

	m.MemLimit = float64(v.MemoryStats.Limit)
	m.NetworkRx, m.NetworkTx = calculateNetwork(v.Networks)

	// 3. Disk I/O: Windows uses StorageStats, Linux uses BlkioStats
	if isWindows {
		m.BlockRead = v.StorageStats.ReadSizeBytes
		m.BlockWrite = v.StorageStats.WriteSizeBytes
	} else {
		m.BlockRead, m.BlockWrite = calculateBlockIO(v.BlkioStats)
	}

	return m
}

func calculateCPUPercentUnix(v *container.StatsResponse) float64 {
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
		return (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}
	return 0.0
}

func calculateCPUPercentWindows(v *container.StatsResponse) float64 {
	// Max number of 100ns intervals between the previous time read and now
	possIntervals := uint64(v.Read.Sub(v.PreRead).Nanoseconds()) // Start with number of ns intervals
	possIntervals /= 100                                         // Convert to number of 100ns intervals
	possIntervals *= uint64(v.NumProcs)                          // Multiply by the number of processors

	// Percentage avoiding divide-by-zero
	if possIntervals > 0 && v.CPUStats.CPUUsage.TotalUsage >= v.PreCPUStats.CPUUsage.TotalUsage {
		intervalsUsed := v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage
		return float64(intervalsUsed) / float64(possIntervals) * 100.0
	}
	return 0.00
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
