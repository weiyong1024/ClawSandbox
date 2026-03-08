package container

import (
	"fmt"

	docker "github.com/fsouza/go-dockerclient"
)

// ContainerStats holds a snapshot of container resource usage.
type ContainerStats struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryUsage int64   `json:"memory_usage"`
	MemoryLimit int64   `json:"memory_limit"`
}

// Stats returns a point-in-time resource usage snapshot for the given container.
func Stats(cli *docker.Client, containerID string) (*ContainerStats, error) {
	errCh := make(chan error, 1)
	statsCh := make(chan *docker.Stats, 1)

	done := make(chan bool)
	go func() {
		errCh <- cli.Stats(docker.StatsOptions{
			ID:     containerID,
			Stats:  statsCh,
			Stream: false,
			Done:   done,
		})
	}()

	raw, ok := <-statsCh
	close(done)

	if !ok {
		select {
		case err := <-errCh:
			if err != nil {
				return nil, fmt.Errorf("stats for %s: %w", containerID, err)
			}
		default:
		}
		return nil, fmt.Errorf("stats for %s: no data returned", containerID)
	}

	// Wait for Stats call to finish.
	if err := <-errCh; err != nil {
		return nil, fmt.Errorf("stats for %s: %w", containerID, err)
	}

	return parseStats(raw), nil
}

func parseStats(raw *docker.Stats) *ContainerStats {
	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(raw.CPUStats.SystemCPUUsage - raw.PreCPUStats.SystemCPUUsage)

	cpuPercent := 0.0
	if sysDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / sysDelta) * float64(raw.CPUStats.OnlineCPUs) * 100.0
	}

	return &ContainerStats{
		CPUPercent:  cpuPercent,
		MemoryUsage: int64(raw.MemoryStats.Usage),
		MemoryLimit: int64(raw.MemoryStats.Limit),
	}
}
