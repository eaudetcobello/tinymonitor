// Package collector samples host resource metrics.
package collector

import (
	"errors"
	"fmt"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/model"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// Value names of the CPU metric.
const (
	usedPct   = "used_pct"
	userPct   = "user_pct"
	systemPct = "system_pct"
)

const (
	bytesPerGiB = 1024 * 1024 * 1024
	// agentID is the agent every sample is recorded under until agents are configurable.
	agentID uint32 = 123123123
)

var (
	// ErrNoBaseline is returned by the first CPU call, which only records the reading later calls compare against.
	ErrNoBaseline = errors.New("cpu: no baseline reading yet")
	errNoCPUTime  = errors.New("cpu: no time elapsed between readings")
	errNoCPUTimes = errors.New("cpu times: empty result")
)

type Collector interface {
	Memory() (model.Metric, error)
	CPU() (model.Metric, error)
}

// Gopsutil reads host metrics with gopsutil. CPU usage is the change between two readings, so it keeps the
// previous one; use it through a pointer from a single goroutine.
type Gopsutil struct {
	prevCPU *cpu.TimesStat
}

func (*Gopsutil) Memory() (model.Metric, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return model.Metric{}, fmt.Errorf("virtual memory: %w", err)
	}

	return model.Metric{
		AgentID: agentID,
		Time:    time.Now().UTC(),
		Kind:    model.KindMemory,
		Values: map[string]float64{
			"total_gib": float64(v.Total) / bytesPerGiB,
			"avail_gib": float64(v.Available) / bytesPerGiB,
			usedPct:     v.UsedPercent,
		},
	}, nil
}

// CPU returns host-wide CPU usage since the previous call. The first call returns ErrNoBaseline.
func (g *Gopsutil) CPU() (model.Metric, error) {
	times, err := cpu.Times(false)
	if err != nil {
		return model.Metric{}, fmt.Errorf("cpu times: %w", err)
	}

	if len(times) == 0 {
		return model.Metric{}, errNoCPUTimes
	}

	cur := times[0]
	prev := g.prevCPU
	g.prevCPU = &cur

	if prev == nil {
		return model.Metric{}, ErrNoBaseline
	}

	values, err := cpuUsage(*prev, cur)
	if err != nil {
		return model.Metric{}, err
	}

	return model.Metric{
		AgentID: agentID,
		Time:    time.Now().UTC(),
		Kind:    model.KindCPU,
		Values:  values,
	}, nil
}

// cpuUsage returns the share of CPU time spent busy, in user mode and in system mode between two readings, in percent.
// Idle and iowait count as not busy. Guest time is already part of user time, so it is left out of the total.
func cpuUsage(prev, cur cpu.TimesStat) (map[string]float64, error) {
	total := cpuTotal(cur) - cpuTotal(prev)
	if total <= 0 {
		return nil, errNoCPUTime
	}

	idle := (cur.Idle + cur.Iowait) - (prev.Idle + prev.Iowait)

	return map[string]float64{
		usedPct:   pct(total-idle, total),
		userPct:   pct(cur.User-prev.User, total),
		systemPct: pct(cur.System-prev.System, total),
	}, nil
}

func cpuTotal(t cpu.TimesStat) float64 {
	return t.User + t.System + t.Idle + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal
}

// pct returns part as a percentage of total, clamped to [0, 100] against counter jitter.
func pct(part, total float64) float64 {
	return min(max(part/total*100, 0), 100)
}
