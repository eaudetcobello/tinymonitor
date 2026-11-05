package model

import (
	"time"
)

type Kind string

const (
	KindMemory Kind = "memory"
	KindCPU    Kind = "cpu"
)

type Metric struct {
	AgentID uint32
	SortKey string
	Kind    Kind
	Time    time.Time
	Values  map[string]float64
}
