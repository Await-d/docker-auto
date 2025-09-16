package monitoring

import (
	"context"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// MetricsCollector manages all metrics collection
type MetricsCollector struct {
	mu                sync.RWMutex
	counters          map[string]*Counter
	gauges            map[string]*Gauge
	histograms        map[string]*Histogram
	summaries         map[string]*Summary
	config            MetricsConfig
	updateChan        chan MetricUpdate
	done              chan bool
	systemCollector   *SystemMetricsCollector
	componentCollectors map[string]ComponentCollector
}

// ComponentCollector interface for component-specific metric collection
type ComponentCollector interface {
	CollectMetrics() (ComponentMetrics, error)
	Name() string
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config MetricsConfig) *MetricsCollector {
	mc := &MetricsCollector{
		counters:            make(map[string]*Counter),
		gauges:              make(map[string]*Gauge),
		histograms:          make(map[string]*Histogram),
		summaries:           make(map[string]*Summary),
		config:              config,
		updateChan:          make(chan MetricUpdate, config.BufferSize),
		done:                make(chan bool),
		systemCollector:     NewSystemMetricsCollector(),
		componentCollectors: make(map[string]ComponentCollector),
	}

	if config.Enabled {
		go mc.processingLoop()
		go mc.collectionLoop()
	}

	return mc
}

// RegisterCounter creates and registers a new counter
func (mc *MetricsCollector) RegisterCounter(name, description string, labels map[string]string) *Counter {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.metricKey(name, labels)
	if existing, exists := mc.counters[key]; exists {
		return existing
	}

	counter := NewCounter(name, description, labels)
	mc.counters[key] = counter
	return counter
}

// RegisterGauge creates and registers a new gauge
func (mc *MetricsCollector) RegisterGauge(name, description string, labels map[string]string) *Gauge {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.metricKey(name, labels)
	if existing, exists := mc.gauges[key]; exists {
		return existing
	}

	gauge := NewGauge(name, description, labels)
	mc.gauges[key] = gauge
	return gauge
}

// RegisterHistogram creates and registers a new histogram
func (mc *MetricsCollector) RegisterHistogram(name, description string, buckets []float64, labels map[string]string) *Histogram {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.metricKey(name, labels)
	if existing, exists := mc.histograms[key]; exists {
		return existing
	}

	if buckets == nil {
		buckets = DefaultHistogramBuckets()
	}

	histogram := NewHistogram(name, description, buckets, labels)
	mc.histograms[key] = histogram
	return histogram
}

// RegisterSummary creates and registers a new summary
func (mc *MetricsCollector) RegisterSummary(name, description string, quantiles []float64, maxAge time.Duration, labels map[string]string) *Summary {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.metricKey(name, labels)
	if existing, exists := mc.summaries[key]; exists {
		return existing
	}

	if quantiles == nil {
		quantiles = DefaultQuantiles()
	}

	summary := NewSummary(name, description, quantiles, maxAge, labels)
	mc.summaries[key] = summary
	return summary
}

// RegisterComponentCollector registers a component-specific collector
func (mc *MetricsCollector) RegisterComponentCollector(collector ComponentCollector) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.componentCollectors[collector.Name()] = collector
}

// UpdateMetric sends a metric update through the channel
func (mc *MetricsCollector) UpdateMetric(update MetricUpdate) {
	if !mc.config.Enabled {
		return
	}

	select {
	case mc.updateChan <- update:
	default:
		// Buffer full, drop metric
	}
}

// GetAllMetrics returns all current metrics
func (mc *MetricsCollector) GetAllMetrics() []Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var metrics []Metric

	// Collect counters
	for _, counter := range mc.counters {
		metrics = append(metrics, counter.ToMetric())
	}

	// Collect gauges
	for _, gauge := range mc.gauges {
		metrics = append(metrics, gauge.ToMetric())
	}

	// Collect histograms
	for _, histogram := range mc.histograms {
		metrics = append(metrics, histogram.ToMetric())
	}

	// Collect summaries
	for _, summary := range mc.summaries {
		metrics = append(metrics, summary.ToMetric())
	}

	return metrics
}

// GetSystemMetrics returns current system metrics
func (mc *MetricsCollector) GetSystemMetrics() (SystemMetrics, error) {
	return mc.systemCollector.Collect()
}

// GetComponentMetrics returns metrics for all registered components
func (mc *MetricsCollector) GetComponentMetrics() []ComponentMetrics {
	mc.mu.RLock()
	collectors := make([]ComponentCollector, 0, len(mc.componentCollectors))
	for _, collector := range mc.componentCollectors {
		collectors = append(collectors, collector)
	}
	mc.mu.RUnlock()

	var componentMetrics []ComponentMetrics
	for _, collector := range collectors {
		if metrics, err := collector.CollectMetrics(); err == nil {
			componentMetrics = append(componentMetrics, metrics)
		}
	}

	return componentMetrics
}

// processingLoop processes metric updates
func (mc *MetricsCollector) processingLoop() {
	for {
		select {
		case update := <-mc.updateChan:
			mc.processMetricUpdate(update)
		case <-mc.done:
			return
		}
	}
}

// collectionLoop periodically collects system metrics
func (mc *MetricsCollector) collectionLoop() {
	ticker := time.NewTicker(mc.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.collectSystemMetrics()
		case <-mc.done:
			return
		}
	}
}

// processMetricUpdate processes a single metric update
func (mc *MetricsCollector) processMetricUpdate(update MetricUpdate) {
	key := mc.metricKey(update.Name, update.Labels)

	switch update.Type {
	case MetricTypeCounter:
		counter := mc.getOrCreateCounter(update.Name, "", update.Labels)
		switch update.Operation {
		case "inc":
			counter.Inc()
		case "add":
			counter.Add(update.Value)
		}

	case MetricTypeGauge:
		gauge := mc.getOrCreateGauge(update.Name, "", update.Labels)
		switch update.Operation {
		case "set":
			gauge.Set(update.Value)
		case "inc":
			gauge.Inc()
		case "dec":
			gauge.Dec()
		case "add":
			gauge.Add(update.Value)
		case "sub":
			gauge.Sub(update.Value)
		}

	case MetricTypeHistogram:
		histogram := mc.getOrCreateHistogram(update.Name, "", nil, update.Labels)
		if update.Operation == "observe" {
			histogram.Observe(update.Value)
		}

	case MetricTypeSummary:
		summary := mc.getOrCreateSummary(update.Name, "", nil, time.Hour, update.Labels)
		if update.Operation == "observe" {
			summary.Observe(update.Value)
		}
	}
}

// collectSystemMetrics collects and stores system metrics
func (mc *MetricsCollector) collectSystemMetrics() {
	systemMetrics, err := mc.systemCollector.Collect()
	if err != nil {
		return
	}

	// Update gauges with system metrics
	mc.updateSystemMetricGauges(systemMetrics)
}

// updateSystemMetricGauges updates gauges with system metric values
func (mc *MetricsCollector) updateSystemMetricGauges(metrics SystemMetrics) {
	// CPU metrics
	mc.getOrCreateGauge("system_cpu_usage_percent", "CPU usage percentage", nil).Set(metrics.CPU.Usage)
	mc.getOrCreateGauge("system_cpu_load_avg_1", "CPU load average 1 minute", nil).Set(metrics.CPU.LoadAvg1)
	mc.getOrCreateGauge("system_cpu_load_avg_5", "CPU load average 5 minutes", nil).Set(metrics.CPU.LoadAvg5)
	mc.getOrCreateGauge("system_cpu_load_avg_15", "CPU load average 15 minutes", nil).Set(metrics.CPU.LoadAvg15)

	// Memory metrics
	mc.getOrCreateGauge("system_memory_total_bytes", "Total memory in bytes", nil).Set(float64(metrics.Memory.Total))
	mc.getOrCreateGauge("system_memory_used_bytes", "Used memory in bytes", nil).Set(float64(metrics.Memory.Used))
	mc.getOrCreateGauge("system_memory_available_bytes", "Available memory in bytes", nil).Set(float64(metrics.Memory.Available))
	mc.getOrCreateGauge("system_memory_usage_percent", "Memory usage percentage", nil).Set(metrics.Memory.Usage)

	// Disk metrics
	mc.getOrCreateGauge("system_disk_total_bytes", "Total disk space in bytes", nil).Set(float64(metrics.Disk.Total))
	mc.getOrCreateGauge("system_disk_used_bytes", "Used disk space in bytes", nil).Set(float64(metrics.Disk.Used))
	mc.getOrCreateGauge("system_disk_available_bytes", "Available disk space in bytes", nil).Set(float64(metrics.Disk.Available))
	mc.getOrCreateGauge("system_disk_usage_percent", "Disk usage percentage", nil).Set(metrics.Disk.Usage)

	// Network metrics
	mc.getOrCreateGauge("system_network_bytes_received", "Network bytes received", nil).Set(float64(metrics.Network.BytesReceived))
	mc.getOrCreateGauge("system_network_bytes_sent", "Network bytes sent", nil).Set(float64(metrics.Network.BytesSent))

	// Docker metrics
	mc.getOrCreateGauge("docker_containers_running", "Running Docker containers", nil).Set(float64(metrics.Docker.ContainersRunning))
	mc.getOrCreateGauge("docker_containers_stopped", "Stopped Docker containers", nil).Set(float64(metrics.Docker.ContainersStopped))
	mc.getOrCreateGauge("docker_images_total", "Total Docker images", nil).Set(float64(metrics.Docker.Images))

	// Application metrics
	mc.getOrCreateGauge("app_uptime_seconds", "Application uptime in seconds", nil).Set(metrics.Application.Uptime.Seconds())
	mc.getOrCreateGauge("app_active_connections", "Active connections", nil).Set(float64(metrics.Application.ActiveConnections))
}

// Helper methods to get or create metrics
func (mc *MetricsCollector) getOrCreateCounter(name, description string, labels map[string]string) *Counter {
	key := mc.metricKey(name, labels)
	if counter, exists := mc.counters[key]; exists {
		return counter
	}
	return mc.RegisterCounter(name, description, labels)
}

func (mc *MetricsCollector) getOrCreateGauge(name, description string, labels map[string]string) *Gauge {
	key := mc.metricKey(name, labels)
	if gauge, exists := mc.gauges[key]; exists {
		return gauge
	}
	return mc.RegisterGauge(name, description, labels)
}

func (mc *MetricsCollector) getOrCreateHistogram(name, description string, buckets []float64, labels map[string]string) *Histogram {
	key := mc.metricKey(name, labels)
	if histogram, exists := mc.histograms[key]; exists {
		return histogram
	}
	return mc.RegisterHistogram(name, description, buckets, labels)
}

func (mc *MetricsCollector) getOrCreateSummary(name, description string, quantiles []float64, maxAge time.Duration, labels map[string]string) *Summary {
	key := mc.metricKey(name, labels)
	if summary, exists := mc.summaries[key]; exists {
		return summary
	}
	return mc.RegisterSummary(name, description, quantiles, maxAge, labels)
}

// metricKey generates a unique key for a metric based on name and labels
func (mc *MetricsCollector) metricKey(name string, labels map[string]string) string {
	key := name
	if len(labels) > 0 {
		for k, v := range labels {
			key += "|" + k + "=" + v
		}
	}
	return key
}

// Close stops the metrics collector
func (mc *MetricsCollector) Close() error {
	close(mc.done)
	return nil
}

// TimeDuration is a helper method to time function execution
func (mc *MetricsCollector) TimeDuration(name string, labels map[string]string, fn func()) {
	start := time.Now()
	fn()
	duration := time.Since(start)

	histogram := mc.getOrCreateHistogram(name+"_duration_seconds", "Duration in seconds", DefaultHistogramBuckets(), labels)
	histogram.Observe(duration.Seconds())
}

// TimeDurationWithResult is a helper method to time function execution with result
func TimeDurationWithResult[T any](mc *MetricsCollector, name string, labels map[string]string, fn func() T) T {
	start := time.Now()
	result := fn()
	duration := time.Since(start)

	histogram := mc.getOrCreateHistogram(name+"_duration_seconds", "Duration in seconds", DefaultHistogramBuckets(), labels)
	histogram.Observe(duration.Seconds())

	return result
}

// RecordError records an error occurrence
func (mc *MetricsCollector) RecordError(component string, errorType string, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["component"] = component
	labels["error_type"] = errorType

	counter := mc.getOrCreateCounter("errors_total", "Total number of errors", labels)
	counter.Inc()
}

// RecordOperation records an operation (success or failure)
func (mc *MetricsCollector) RecordOperation(operation string, success bool, duration time.Duration, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["operation"] = operation

	// Record operation count
	opLabels := CloneLabels(labels)
	if success {
		opLabels["status"] = "success"
	} else {
		opLabels["status"] = "failure"
	}

	counter := mc.getOrCreateCounter("operations_total", "Total number of operations", opLabels)
	counter.Inc()

	// Record operation duration
	histogram := mc.getOrCreateHistogram("operation_duration_seconds", "Operation duration in seconds", DefaultHistogramBuckets(), labels)
	histogram.Observe(duration.Seconds())
}