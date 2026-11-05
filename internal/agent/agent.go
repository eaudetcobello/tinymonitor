package agent

import (
	"context"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/collector"
	"github.com/eaudetcobello/tinymonitor/internal/model"
)

type Sink interface {
	Put(ctx context.Context, m model.Metric)
}

type Job struct {
	Name     string
	Interval time.Duration
	Collect  func(model.Metric, error)
}

type Agent struct {
	ID        uint32
	Collector collector.Collector
	Sink      Sink
	Jobs      []Job
}
