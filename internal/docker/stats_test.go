package docker

import (
	"testing"

	"github.com/moby/moby/api/types/container"
)

func TestCalculateStats(t *testing.T) {
	// 1. Nil stats response
	zeroMetrics := CalculateStats(nil)
	if zeroMetrics.CPUPercent != 0 || zeroMetrics.MemUsage != 0 {
		t.Errorf("CalculateStats(nil) should return zero metrics, got %+v", zeroMetrics)
	}

	// 2. Realistic Docker stats response
	stats := &container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage: container.CPUUsage{
				TotalUsage:  200000000,
				PercpuUsage: []uint64{100000000, 100000000},
			},
			SystemUsage: 2000000000,
			OnlineCPUs:  2,
		},
		PreCPUStats: container.CPUStats{
			CPUUsage: container.CPUUsage{
				TotalUsage: 100000000,
			},
			SystemUsage: 1000000000,
		},
		MemoryStats: container.MemoryStats{
			Usage: 50000000,
			Limit: 100000000,
			Stats: map[string]uint64{
				"inactive_file": 10000000, // cgroup v2 cache to be deducted
			},
		},
		Networks: map[string]container.NetworkStats{
			"eth0": {RxBytes: 1500, TxBytes: 2500},
			"eth1": {RxBytes: 500, TxBytes: 500},
		},
		BlkioStats: container.BlkioStats{
			IoServiceBytesRecursive: []container.BlkioStatEntry{
				{Op: "Read", Value: 4096},
				{Op: "Write", Value: 8192},
			},
		},
	}

	metrics := CalculateStats(stats)

	// CPU = (100M / 1000M) * 2 * 100 = 20.0%
	if metrics.CPUPercent < 19.9 || metrics.CPUPercent > 20.1 {
		t.Errorf("CalculateStats CPUPercent = %f, want ~20.0", metrics.CPUPercent)
	}

	// MemUsage = 50M - 10M = 40M
	if metrics.MemUsage != 40000000 {
		t.Errorf("CalculateStats MemUsage = %f, want 40000000", metrics.MemUsage)
	}

	// Networks
	if metrics.NetworkRx != 2000 || metrics.NetworkTx != 3000 {
		t.Errorf("CalculateStats Networks Rx=%f Tx=%f, want 2000 and 3000", metrics.NetworkRx, metrics.NetworkTx)
	}

	// Block IO
	if metrics.BlockRead != 4096 || metrics.BlockWrite != 8192 {
		t.Errorf("CalculateStats BlockIO Read=%d Write=%d, want 4096 and 8192", metrics.BlockRead, metrics.BlockWrite)
	}
}

func Test_calculateNetwork(t *testing.T) {
	networks := map[string]container.NetworkStats{
		"eth0": {RxBytes: 100, TxBytes: 200},
		"eth1": {RxBytes: 300, TxBytes: 400},
	}
	rx, tx := calculateNetwork(networks)
	if rx != 400 || tx != 600 {
		t.Errorf("calculateNetwork() = (%f, %f), want (400, 600)", rx, tx)
	}
}

func Test_calculateBlockIO(t *testing.T) {
	blkio := container.BlkioStats{
		IoServiceBytesRecursive: []container.BlkioStatEntry{
			{Op: "r", Value: 1000},
			{Op: "R", Value: 2000},
			{Op: "w", Value: 3000},
			{Op: "W", Value: 4000},
			{Op: "", Value: 500}, // Empty Op should be skipped
		},
	}
	read, write := calculateBlockIO(blkio)
	if read != 3000 || write != 7000 {
		t.Errorf("calculateBlockIO() = (%d, %d), want (3000, 7000)", read, write)
	}
}
