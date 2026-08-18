// Package metrics collects counters and histograms and renders them in the Prometheus
// text format. There is no package-level registry: whoever records is given one.
package metrics

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
)

// bucketCount is how many upper bounds every histogram here carries.
const bucketCount = 8

// buckets returns the upper bounds in seconds. It is a function, not a package variable:
// a global slice of bounds is state anyone could rewrite at runtime.
func buckets() [bucketCount]float64 {
	return [bucketCount]float64{0.005, 0.025, 0.1, 0.5, 1, 2.5, 5, 10}
}

// Label is one dimension of a series, kept ordered so the output is stable.
type Label struct {
	Name  string
	Value string
}

type seriesKey struct {
	name   string
	labels string
}

type histogram struct {
	buckets [bucketCount]uint64
	sum     float64
	count   uint64
}

// Registry holds every series of the process; safe for concurrent use.
type Registry struct {
	mu         sync.Mutex
	counters   map[seriesKey]uint64
	histograms map[seriesKey]*histogram
	help       map[string]string
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		counters:   make(map[seriesKey]uint64),
		histograms: make(map[seriesKey]*histogram),
		help:       make(map[string]string),
	}
}

// Describe records the HELP line of a metric, so a series that never fired still explains
// itself when someone reads /metrics.
func (r *Registry) Describe(name, help string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.help[name] = help
}

// Count adds one to the counter.
func (r *Registry) Count(name string, labels ...Label) {
	key := seriesKey{name: name, labels: encode(labels)}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.counters[key]++
}

// Observe records a value in seconds.
func (r *Registry) Observe(name string, seconds float64, labels ...Label) {
	key := seriesKey{name: name, labels: encode(labels)}

	r.mu.Lock()
	defer r.mu.Unlock()

	h, ok := r.histograms[key]
	if !ok {
		h = &histogram{}
		r.histograms[key] = h
	}

	h.count++
	h.sum += seconds

	for i, upper := range buckets() {
		if seconds <= upper {
			h.buckets[i]++
		}
	}
}

// Render writes every series in the Prometheus text format.
func (r *Registry) Render() string {
	r.mu.Lock()
	counters := maps.Clone(r.counters)
	histograms := make(map[seriesKey]histogram, len(r.histograms))

	for k, h := range r.histograms {
		histograms[k] = *h
	}
	help := maps.Clone(r.help)
	r.mu.Unlock()

	var body strings.Builder

	for _, k := range sorted(maps.Keys(counters)) {
		writeHelp(&body, help, k.name, "counter")
		fmt.Fprintf(&body, "%s%s %d\n", k.name, k.labels, counters[k])
	}

	for _, k := range sorted(maps.Keys(histograms)) {
		h := histograms[k]

		writeHelp(&body, help, k.name, "histogram")

		for i, upper := range buckets() {
			fmt.Fprintf(&body, "%s_bucket%s %d\n", k.name, withLE(k.labels, fmt.Sprintf("%g", upper)), h.buckets[i])
		}

		fmt.Fprintf(&body, "%s_bucket%s %d\n", k.name, withLE(k.labels, "+Inf"), h.count)
		fmt.Fprintf(&body, "%s_sum%s %g\n", k.name, k.labels, h.sum)
		fmt.Fprintf(&body, "%s_count%s %d\n", k.name, k.labels, h.count)
	}

	return body.String()
}

func writeHelp(body *strings.Builder, help map[string]string, name, kind string) {
	if strings.Contains(body.String(), "# TYPE "+name+" ") {
		return
	}
	if text, ok := help[name]; ok {
		fmt.Fprintf(body, "# HELP %s %s\n", name, text)
	}

	fmt.Fprintf(body, "# TYPE %s %s\n", name, kind)
}

func sorted(keys iter.Seq[seriesKey]) []seriesKey {
	return slices.SortedFunc(keys, func(a, b seriesKey) int {
		return cmp.Or(cmp.Compare(a.name, b.name), cmp.Compare(a.labels, b.labels))
	})
}

func encode(labels []Label) string {
	if len(labels) == 0 {
		return ""
	}

	parts := make([]string, 0, len(labels))
	for _, l := range labels {
		parts = append(parts, fmt.Sprintf("%s=%q", l.Name, l.Value))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

func withLE(labels, le string) string {
	if labels == "" {
		return fmt.Sprintf("{le=%q}", le)
	}

	return labels[:len(labels)-1] + fmt.Sprintf(",le=%q}", le)
}
