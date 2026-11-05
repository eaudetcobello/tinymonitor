package tui

import (
	"slices"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/model"
)

// windowPresets returns the windows the widen and narrow keys step through, shortest first.
func windowPresets() []time.Duration {
	return []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour}
}

// widerWindow returns the shortest preset longer than cur, or cur when none is.
func widerWindow(cur time.Duration) time.Duration {
	for _, w := range windowPresets() {
		if w > cur {
			return w
		}
	}

	return cur
}

// narrowerWindow returns the longest preset shorter than cur and at least refresh, or cur when none is.
func narrowerWindow(cur, refresh time.Duration) time.Duration {
	for _, w := range slices.Backward(windowPresets()) {
		if w < cur && w >= refresh {
			return w
		}
	}

	return cur
}

// nextValue returns the name after cur in available, wrapping around. When cur is not in available it returns the
// first name; when available is empty it returns cur.
func nextValue(cur string, available []string) string {
	if len(available) == 0 {
		return cur
	}

	return available[(slices.Index(available, cur)+1)%len(available)]
}

// nextKind returns the kind after cur in model.Kinds, wrapping around.
func nextKind(cur model.Kind) model.Kind {
	kinds := model.Kinds()

	return kinds[(slices.Index(kinds, cur)+1)%len(kinds)]
}
