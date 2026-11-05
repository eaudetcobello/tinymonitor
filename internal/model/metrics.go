package model

import (
	"errors"
	"fmt"
	"time"
)

type Kind string

const (
	KindMemory Kind = "memory"
	KindCPU    Kind = "cpu"
)

// Kinds returns every known metric kind.
func Kinds() []Kind {
	return []Kind{KindCPU, KindMemory}
}

var (
	ErrMissingKind = errors.New("missing kind")
	ErrUnknownKind = errors.New("unknown kind")
)

// ParseKind returns the Kind named by s, or an error if s is empty or not a known kind.
func ParseKind(s string) (Kind, error) {
	if s == "" {
		return "", ErrMissingKind
	}

	for _, k := range Kinds() {
		if string(k) == s {
			return k, nil
		}
	}

	return "", fmt.Errorf("%w %q, expected one of: %v", ErrUnknownKind, s, Kinds())
}

type Metric struct {
	AgentID uint32             `json:"agent_id"`
	Kind    Kind               `json:"kind"`
	Time    time.Time          `json:"time"`
	Values  map[string]float64 `json:"values"`
}
