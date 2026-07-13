package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/ui/virtual"
)

// PerfMonitor tracks runtime performance metrics for UI operations.
type PerfMonitor struct {
	enabled         bool
	paintThreshold  time.Duration
	scrollThreshold time.Duration
	mu              sync.Mutex
	metrics         map[string]*perfMetric
}

// perfMetric tracks statistics for a single operation type.
type perfMetric struct {
	count     int64
	totalNs   int64
	minNs     int64
	maxNs     int64
	threshold time.Duration
	exceeds   int64
}

// NewPerfMonitor creates a new performance monitor with the given thresholds.
func NewPerfMonitor(paintThreshold, scrollThreshold time.Duration) *PerfMonitor {
	return &PerfMonitor{
		enabled:         true,
		paintThreshold:  paintThreshold,
		scrollThreshold: scrollThreshold,
		metrics:         make(map[string]*perfMetric),
	}
}

// RecordPaint records a paint operation duration.
func (pm *PerfMonitor) RecordPaint(name string, duration time.Duration) {
	if !pm.enabled {
		return
	}

	pm.recordMetric(name+"_paint", duration, pm.paintThreshold)
}

// RecordScroll records a scroll operation duration.
func (pm *PerfMonitor) RecordScroll(name string, duration time.Duration) {
	if !pm.enabled {
		return
	}

	pm.recordMetric(name+"_scroll", duration, pm.scrollThreshold)
}

// Report generates a performance summary string.
func (pm *PerfMonitor) Report() string {
	if !pm.enabled {
		return "Performance monitoring is disabled."
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.metrics) == 0 {
		return "No performance data collected yet."
	}

	var report string

	report += "=== Performance Report ===\n"

	// Group by operation type
	paintMetrics := make(map[string]*perfMetric)
	scrollMetrics := make(map[string]*perfMetric)

	for key, m := range pm.metrics {
		if len(key) > 6 && key[len(key)-6:] == "_paint" {
			name := key[:len(key)-6]
			paintMetrics[name] = m
		} else if len(key) > 7 && key[len(key)-7:] == "_scroll" {
			name := key[:len(key)-7]
			scrollMetrics[name] = m
		}
	}

	// Paint section
	if len(paintMetrics) > 0 {
		report += "\nPaint Operations (threshold: " + pm.paintThreshold.String() + "):\n"
		for name, m := range paintMetrics {
			report += fmt.Sprintf("  %s: %.2fµs avg (%.2fµs min, %.2fµs max), %d ops, %d exceeds\n",
				name,
				float64(m.totalNs)/float64(m.count)/1000,
				float64(m.minNs)/1000,
				float64(m.maxNs)/1000,
				m.count,
				m.exceeds)
		}
	}

	// Scroll section
	if len(scrollMetrics) > 0 {
		report += "\nScroll Operations (threshold: " + pm.scrollThreshold.String() + "):\n"
		for name, m := range scrollMetrics {
			report += fmt.Sprintf("  %s: %.2fµs avg (%.2fµs min, %.2fµs max), %d ops, %d exceeds\n",
				name,
				float64(m.totalNs)/float64(m.count)/1000,
				float64(m.minNs)/1000,
				float64(m.maxNs)/1000,
				m.count,
				m.exceeds)
		}
	}

	return report
}

// Summary returns a brief summary for the status bar.
func (pm *PerfMonitor) Summary() string {
	if !pm.enabled {
		return ""
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	var (
		totalOps     int64
		totalExceeds int64
	)

	for _, m := range pm.metrics {
		totalOps += m.count
		totalExceeds += m.exceeds
	}

	if totalOps == 0 {
		return "Perf: collecting..."
	}

	return fmt.Sprintf("Perf: %d ops, %d exceeds threshold", totalOps, totalExceeds)
}

func (pm *PerfMonitor) recordMetric(key string, duration time.Duration, threshold time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	m, ok := pm.metrics[key]
	if !ok {
		m = &perfMetric{
			minNs:     int64(duration),
			maxNs:     int64(duration),
			threshold: threshold,
		}
		pm.metrics[key] = m
	}

	ns := int64(duration)
	m.count++
	m.totalNs += ns

	if ns < m.minNs {
		m.minNs = ns
	}

	if ns > m.maxNs {
		m.maxNs = ns
	}

	if duration > threshold {
		m.exceeds++
	}
}

// InstrumentedVirtualList wraps a VirtualList to record performance metrics.
type InstrumentedVirtualList struct {
	*virtual.VirtualList

	monitor *PerfMonitor
	name    string
}

// NewInstrumentedVirtualList creates a new instrumented virtual list.
func NewInstrumentedVirtualList(name string, base *virtual.VirtualList, monitor *PerfMonitor) *InstrumentedVirtualList {
	return &InstrumentedVirtualList{
		VirtualList: base,
		monitor:     monitor,
		name:        name,
	}
}

// Paint records paint performance before delegating to the base list.
func (i *InstrumentedVirtualList) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	start := time.Now()

	i.VirtualList.Paint(d, ctx)

	duration := time.Since(start)
	i.monitor.RecordPaint(i.name, duration)
}

// ScrollTo records scroll performance before delegating to the base list.
func (i *InstrumentedVirtualList) ScrollTo(ctx *rovel.Ctx, item int) {
	start := time.Now()

	i.VirtualList.ScrollTo(ctx, item)

	duration := time.Since(start)
	i.monitor.RecordScroll(i.name, duration)
}

// ScrollBy records scroll performance before delegating to the base list.
func (i *InstrumentedVirtualList) ScrollBy(ctx *rovel.Ctx, delta int) {
	start := time.Now()

	i.VirtualList.ScrollBy(ctx, delta)

	duration := time.Since(start)
	i.monitor.RecordScroll(i.name+"_by", duration)
}

// ScrollTop records scroll performance before delegating to the base list.
func (i *InstrumentedVirtualList) ScrollTop(ctx *rovel.Ctx) {
	start := time.Now()

	i.VirtualList.ScrollTop(ctx)

	duration := time.Since(start)
	i.monitor.RecordScroll(i.name+"_top", duration)
}

// ScrollBottom records scroll performance before delegating to the base list.
func (i *InstrumentedVirtualList) ScrollBottom(ctx *rovel.Ctx) {
	start := time.Now()

	i.VirtualList.ScrollBottom(ctx)

	duration := time.Since(start)
	i.monitor.RecordScroll(i.name+"_bottom", duration)
}

// SelectIndex records selection performance before delegating to the base list.
func (i *InstrumentedVirtualList) SelectIndex(ctx *rovel.Ctx, index int) {
	start := time.Now()

	i.VirtualList.SelectIndex(ctx, index)

	duration := time.Since(start)
	i.monitor.RecordScroll(i.name+"_select", duration)
}
