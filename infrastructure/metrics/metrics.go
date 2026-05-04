package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Default is the process-wide metrics sink (MVP in-memory).
var Default = NewRegistry()

// Registry holds lightweight counters and latency aggregates for text export.
type Registry struct {
	mu     sync.Mutex
	counts map[string]int64
	sum    map[string]float64
}

func NewRegistry() *Registry {
	return &Registry{
		counts: make(map[string]int64),
		sum:    make(map[string]float64),
	}
}

func (r *Registry) Inc(name string, delta int64) {
	if r == nil || name == "" || delta == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counts[name] += delta
}

func (r *Registry) AddLatencySample(name string, ms float64) {
	if r == nil || name == "" || ms < 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sum[name+"_ms_sum"] += ms
	r.counts[name+"_ms_count"]++
}

// ExportText returns one metric per line: name value
func (r *Registry) ExportText() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	keys := make([]string, 0, len(r.counts)+len(r.sum))
	seen := make(map[string]struct{}, len(r.counts)+len(r.sum))
	for k := range r.counts {
		keys = append(keys, k)
		seen[k] = struct{}{}
	}
	for k := range r.sum {
		if _, ok := seen[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		if v, ok := r.sum[k]; ok {
			fmt.Fprintf(&b, "%s %g\n", k, v)
			continue
		}
		fmt.Fprintf(&b, "%s %d\n", k, r.counts[k])
	}
	return b.String()
}

// Inc increments Default.
func Inc(name string, delta int64) { Default.Inc(name, delta) }

// AddLatencySample records evaluate-style latency on Default.
func AddLatencySample(name string, ms float64) { Default.AddLatencySample(name, ms) }

// ExportText exports Default.
func ExportText() string { return Default.ExportText() }
