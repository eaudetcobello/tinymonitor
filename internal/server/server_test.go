package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/model"
	"github.com/eaudetcobello/tinymonitor/internal/server"
)

const (
	validFrom = "2026-11-04T10:00:00Z"
	validTo   = "2026-11-04T10:01:00Z"
)

// fakeReader returns a fixed result and records the arguments of its last call.
type fakeReader struct {
	metrics []model.Metric
	called  bool
	agentID uint32
	kind    model.Kind
}

func (f *fakeReader) Get(_ context.Context, agentID uint32, kind model.Kind, _, _ time.Time) ([]model.Metric, error) {
	f.called = true
	f.agentID = agentID
	f.kind = kind

	return f.metrics, nil
}

// query builds /metrics query parameters, leaving out empty values.
func query(agentID, kind, from, to string) url.Values {
	q := url.Values{}

	for name, value := range map[string]string{"agent_id": agentID, "kind": kind, "from": from, "to": to} {
		if value != "" {
			q.Set(name, value)
		}
	}

	return q
}

func get(t *testing.T, reader server.MetricsReader, query url.Values) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics?"+query.Encode(), nil)
	rec := httptest.NewRecorder()
	server.New(reader).ServeHTTP(rec, req)

	return rec
}

func TestGetMetricsValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    url.Values
		wantBody string
	}{
		{"missing kind", query("", "", validFrom, validTo), "missing kind"},
		{"unknown kind", query("", "disk", validFrom, validTo), `unknown kind "disk"`},
		{"missing from", query("", "cpu", "", validTo), "invalid from timestamp"},
		{"invalid to", query("", "cpu", validFrom, "yesterday"), "invalid to timestamp"},
		{"from after to", query("", "cpu", validTo, validFrom), "from must not be after to"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := &fakeReader{}
			rec := get(t, reader, tt.query)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}

			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body = %q, want it to contain %q", rec.Body.String(), tt.wantBody)
			}

			if reader.called {
				t.Error("reader was called for an invalid request")
			}
		})
	}
}

func TestGetMetricsEmptyRange(t *testing.T) {
	t.Parallel()

	rec := get(t, &fakeReader{metrics: []model.Metric{}}, query("7", "memory", validFrom, validFrom))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("body = %q, want []", got)
	}
}

func TestGetMetricsReturnsSnakeCaseJSON(t *testing.T) {
	t.Parallel()

	reader := &fakeReader{metrics: []model.Metric{{
		AgentID: 7,
		Kind:    model.KindMemory,
		Time:    time.Date(2026, 11, 4, 10, 0, 30, 500, time.UTC),
		Values:  map[string]float64{"used_pct": 42.5},
	}}}

	rec := get(t, reader, query("7", "memory", validFrom, validTo))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if reader.agentID != 7 || reader.kind != model.KindMemory {
		t.Errorf("reader got agent %d kind %q, want 7 memory", reader.agentID, reader.kind)
	}

	var body []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	want := map[string]any{
		"agent_id": float64(7),
		"kind":     "memory",
		"time":     "2026-11-04T10:00:30.0000005Z",
		"values":   map[string]any{"used_pct": 42.5},
	}

	if len(body) != 1 {
		t.Fatalf("got %d metrics, want 1", len(body))
	}

	for key, wantValue := range want {
		if got, ok := body[0][key]; !ok || !jsonEqual(got, wantValue) {
			t.Errorf("%s = %v, want %v", key, got, wantValue)
		}
	}

	if len(body[0]) != len(want) {
		t.Errorf("metric has keys %v, want exactly %d keys", body[0], len(want))
	}
}

func jsonEqual(a, b any) bool {
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)

	return errA == nil && errB == nil && string(ja) == string(jb)
}
