package httpx

import (
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Joaquimgmess/catalog/internal/platform/logging"
)

var durationBuckets = []float64{0.005, 0.025, 0.1, 0.5, 1, 2.5, 5, 10}

type seriesKey struct {
	method string
	route  string
	status int
}

type histogram struct {
	buckets []uint64
	sum     float64
	count   uint64
}

type Metrics struct {
	mu     sync.Mutex
	series map[seriesKey]*histogram
}

func NewMetrics() *Metrics {
	return &Metrics{series: make(map[seriesKey]*histogram)}
}

func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body strings.Builder
		body.WriteString("# HELP http_request_duration_seconds Latency of HTTP requests.\n")
		body.WriteString("# TYPE http_request_duration_seconds histogram\n")

		snapshot, keys := m.snapshot()
		for _, k := range keys {
			h := snapshot[k]
			labels := fmt.Sprintf("method=%q,route=%q,status=\"%d\"", k.method, k.route, k.status)

			for i, upper := range durationBuckets {
				fmt.Fprintf(&body, "http_request_duration_seconds_bucket{%s,le=\"%g\"} %d\n", labels, upper, h.buckets[i])
			}
			fmt.Fprintf(&body, "http_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", labels, h.count)
			fmt.Fprintf(&body, "http_request_duration_seconds_sum{%s} %g\n", labels, h.sum)
			fmt.Fprintf(&body, "http_request_duration_seconds_count{%s} %d\n", labels, h.count)
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		if _, err := io.WriteString(w, body.String()); err != nil {
			logging.FromContext(r.Context()).ErrorContext(r.Context(), "writing metrics", slog.Any("error", err))
		}
	}
}

func (m *Metrics) observe(method, route string, status int, elapsed time.Duration) {
	k := seriesKey{method: method, route: route, status: status}

	m.mu.Lock()
	defer m.mu.Unlock()

	h, ok := m.series[k]
	if !ok {
		h = &histogram{buckets: make([]uint64, len(durationBuckets))}
		m.series[k] = h
	}

	seconds := elapsed.Seconds()
	h.count++
	h.sum += seconds
	for i, upper := range durationBuckets {
		if seconds <= upper {
			h.buckets[i]++
		}
	}
}

func (m *Metrics) snapshot() (map[seriesKey]histogram, []seriesKey) {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot := make(map[seriesKey]histogram, len(m.series))
	for k, h := range m.series {
		snapshot[k] = histogram{buckets: slices.Clone(h.buckets), sum: h.sum, count: h.count}
	}

	keys := slices.SortedFunc(maps.Keys(snapshot), func(a, b seriesKey) int {
		return cmp.Or(cmp.Compare(a.route, b.route), cmp.Compare(a.method, b.method), cmp.Compare(a.status, b.status))
	})

	return snapshot, keys
}

func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return "unmatched"
	}

	pattern := rctx.RoutePattern()
	if pattern == "" {
		return "unmatched"
	}

	return pattern
}
