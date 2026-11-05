// Package collector samples host resource metrics.
package collector

import (
	"fmt"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/model"
	"github.com/shirou/gopsutil/v4/mem"
)

const bytesPerGiB = 1024 * 1024 * 1024

type Collector interface {
	Memory() (model.Metric, error)
}

type Gopsutil struct{}

func (Gopsutil) Memory() (model.Metric, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return model.Metric{}, fmt.Errorf("virtual memory: %w", err)
	}

	return model.Metric{
		AgentID: 123123123,
		Time:    time.Now().UTC(),
		Kind:    model.KindMemory,
		Values: map[string]float64{
			"total_gib": float64(v.Total) / bytesPerGiB,
			"avail_gib": float64(v.Available) / bytesPerGiB,
			"used_pct":  v.UsedPercent,
		},
	}, nil
}
