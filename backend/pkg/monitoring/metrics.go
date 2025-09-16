package monitoring

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// NewCounter creates a new counter metric
func NewCounter(name, description string, labels map[string]string) *Counter {
	if labels == nil {
		labels = make(map[string]string)
	}
	return &Counter{
		name:        name,
		description: description,
		labels:      labels,
	}
}

// Inc increments the counter by 1
func (c *Counter) Inc() {
	c.Add(1)
}

// Add adds the given value to the counter
func (c *Counter) Add(value float64) {
	if value < 0 {
		return // Counters can only increase
	}
	c.mu.Lock()
	c.value += value
	c.mu.Unlock()
}

// Value returns the current value of the counter
func (c *Counter) Value() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// Reset resets the counter to zero
func (c *Counter) Reset() {
	c.mu.Lock()
	c.value = 0
	c.mu.Unlock()
}

// ToMetric converts the counter to a Metric
func (c *Counter) ToMetric() Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return Metric{
		Name:        c.name,
		Type:        MetricTypeCounter,
		Value:       c.value,
		Labels:      c.labels,
		Timestamp:   time.Now(),
		Description: c.description,
	}
}

// NewGauge creates a new gauge metric
func NewGauge(name, description string, labels map[string]string) *Gauge {
	if labels == nil {
		labels = make(map[string]string)
	}
	return &Gauge{
		name:        name,
		description: description,
		labels:      labels,
	}
}

// Set sets the gauge to the given value
func (g *Gauge) Set(value float64) {
	g.mu.Lock()
	g.value = value
	g.mu.Unlock()
}

// Inc increments the gauge by 1
func (g *Gauge) Inc() {
	g.Add(1)
}

// Dec decrements the gauge by 1
func (g *Gauge) Dec() {
	g.Sub(1)
}

// Add adds the given value to the gauge
func (g *Gauge) Add(value float64) {
	g.mu.Lock()
	g.value += value
	g.mu.Unlock()
}

// Sub subtracts the given value from the gauge
func (g *Gauge) Sub(value float64) {
	g.mu.Lock()
	g.value -= value
	g.mu.Unlock()
}

// Value returns the current value of the gauge
func (g *Gauge) Value() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

// ToMetric converts the gauge to a Metric
func (g *Gauge) ToMetric() Metric {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return Metric{
		Name:        g.name,
		Type:        MetricTypeGauge,
		Value:       g.value,
		Labels:      g.labels,
		Timestamp:   time.Now(),
		Description: g.description,
	}
}

// NewHistogram creates a new histogram metric
func NewHistogram(name, description string, buckets []float64, labels map[string]string) *Histogram {
	if labels == nil {
		labels = make(map[string]string)
	}

	// Ensure buckets are sorted and include +Inf
	sortedBuckets := make([]float64, len(buckets))
	copy(sortedBuckets, buckets)
	sort.Float64s(sortedBuckets)

	// Add +Inf bucket if not present
	if len(sortedBuckets) == 0 || sortedBuckets[len(sortedBuckets)-1] != math.Inf(1) {
		sortedBuckets = append(sortedBuckets, math.Inf(1))
	}

	return &Histogram{
		name:        name,
		description: description,
		buckets:     sortedBuckets,
		counts:      make([]uint64, len(sortedBuckets)),
		labels:      labels,
	}
}

// Observe adds an observation to the histogram
func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.sum += value
	h.count++

	// Find the appropriate bucket and increment all buckets >= value
	for i, upperBound := range h.buckets {
		if value <= upperBound {
			h.counts[i]++
		}
	}
}

// Count returns the total number of observations
func (h *Histogram) Count() uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.count
}

// Sum returns the sum of all observations
func (h *Histogram) Sum() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sum
}

// Buckets returns the histogram buckets
func (h *Histogram) Buckets() []HistogramBucket {
	h.mu.RLock()
	defer h.mu.RUnlock()

	buckets := make([]HistogramBucket, len(h.buckets))
	for i, upperBound := range h.buckets {
		buckets[i] = HistogramBucket{
			UpperBound: upperBound,
			Count:      h.counts[i],
		}
	}
	return buckets
}

// Quantile calculates the given quantile from the histogram
func (h *Histogram) Quantile(q float64) float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.count == 0 {
		return 0
	}

	targetCount := float64(h.count) * q
	var cumulativeCount uint64

	for i, count := range h.counts {
		cumulativeCount += count
		if float64(cumulativeCount) >= targetCount {
			if i == 0 {
				return h.buckets[i]
			}
			// Linear interpolation within bucket
			prevBucket := float64(0)
			if i > 0 {
				prevBucket = h.buckets[i-1]
			}
			return prevBucket + (h.buckets[i]-prevBucket)*((targetCount-float64(cumulativeCount-count))/float64(count))
		}
	}

	return h.buckets[len(h.buckets)-1]
}

// ToMetric converts the histogram to a Metric
func (h *Histogram) ToMetric() Metric {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return Metric{
		Name:        h.name,
		Type:        MetricTypeHistogram,
		Value:       h.sum,
		Labels:      h.labels,
		Timestamp:   time.Now(),
		Description: h.description,
	}
}

// NewSummary creates a new summary metric
func NewSummary(name, description string, quantiles []float64, maxAge time.Duration, labels map[string]string) *Summary {
	if labels == nil {
		labels = make(map[string]string)
	}
	return &Summary{
		name:        name,
		description: description,
		quantiles:   quantiles,
		maxAge:      maxAge,
		labels:      labels,
	}
}

// Observe adds an observation to the summary
func (s *Summary) Observe(value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.observations = append(s.observations, value)
	s.timestamps = append(s.timestamps, now)
	s.sum += value
	s.count++

	// Clean old observations
	if s.maxAge > 0 {
		cutoff := now.Add(-s.maxAge)
		var validObservations []float64
		var validTimestamps []time.Time
		var validSum float64

		for i, timestamp := range s.timestamps {
			if timestamp.After(cutoff) {
				validObservations = append(validObservations, s.observations[i])
				validTimestamps = append(validTimestamps, timestamp)
				validSum += s.observations[i]
			}
		}

		s.observations = validObservations
		s.timestamps = validTimestamps
		s.sum = validSum
		s.count = uint64(len(validObservations))
	}
}

// Count returns the total number of observations
func (s *Summary) Count() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

// Sum returns the sum of all observations
func (s *Summary) Sum() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sum
}

// Quantile calculates the given quantile from the summary
func (s *Summary) Quantile(q float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.observations) == 0 {
		return 0
	}

	// Create a copy and sort
	sorted := make([]float64, len(s.observations))
	copy(sorted, s.observations)
	sort.Float64s(sorted)

	// Calculate quantile
	index := q * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// Quantiles calculates all configured quantiles
func (s *Summary) Quantiles() map[float64]float64 {
	result := make(map[float64]float64)
	for _, q := range s.quantiles {
		result[q] = s.Quantile(q)
	}
	return result
}

// ToMetric converts the summary to a Metric
func (s *Summary) ToMetric() Metric {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return Metric{
		Name:        s.name,
		Type:        MetricTypeSummary,
		Value:       s.sum,
		Labels:      s.labels,
		Timestamp:   time.Now(),
		Description: s.description,
	}
}

// DefaultHistogramBuckets provides default histogram buckets for response times
func DefaultHistogramBuckets() []float64 {
	return []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0}
}

// DefaultQuantiles provides default quantiles for summaries
func DefaultQuantiles() []float64 {
	return []float64{0.5, 0.9, 0.95, 0.99}
}

// TimerHelper is a helper for timing operations
type TimerHelper struct {
	start time.Time
}

// NewTimer creates a new timer
func NewTimer() *TimerHelper {
	return &TimerHelper{start: time.Now()}
}

// Duration returns the elapsed duration since the timer was created
func (t *TimerHelper) Duration() time.Duration {
	return time.Since(t.start)
}

// ObserveDuration observes the duration in a histogram or summary
func (t *TimerHelper) ObserveDuration(observer interface{}) {
	duration := time.Since(t.start).Seconds()

	switch obs := observer.(type) {
	case *Histogram:
		obs.Observe(duration)
	case *Summary:
		obs.Observe(duration)
	}
}

// CloneLabels creates a copy of a labels map
func CloneLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return make(map[string]string)
	}

	cloned := make(map[string]string, len(labels))
	for k, v := range labels {
		cloned[k] = v
	}
	return cloned
}

// MergeLabels merges multiple label maps, with later maps overriding earlier ones
func MergeLabels(labelMaps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, labels := range labelMaps {
		for k, v := range labels {
			result[k] = v
		}
	}
	return result
}