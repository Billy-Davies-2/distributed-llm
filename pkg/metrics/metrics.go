package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"distributed-llm/pkg/models"
)

var (
	// Node metrics
	nodeResourcesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "distributed_llm_node_resources",
			Help: "Node resource information",
		},
		[]string{"node_id", "resource_type"},
	)

	nodeStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "distributed_llm_node_status",
			Help: "Node status (0=offline, 1=online, 2=busy, 3=unknown)",
		},
		[]string{"node_id"},
	)

	nodeUptimeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "distributed_llm_node_uptime_seconds",
			Help: "Node uptime in seconds",
		},
		[]string{"node_id"},
	)

	// Layer management metrics
	layersAllocatedGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "distributed_llm_layers_allocated",
			Help: "Number of layers allocated on each node",
		},
		[]string{"node_id", "model_id"},
	)

	layersCapacityGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "distributed_llm_layers_capacity",
			Help: "Maximum layer capacity of each node",
		},
		[]string{"node_id"},
	)

	// Network metrics
	networkMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "distributed_llm_network_messages_total",
			Help: "Total number of network messages sent/received",
		},
		[]string{"node_id", "direction", "message_type"},
	)

	networkLatencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "distributed_llm_network_latency_seconds",
			Help:    "Network request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"node_id", "target_node", "operation"},
	)

	networkConnectionsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "distributed_llm_network_connections",
			Help: "Number of active network connections",
		},
		[]string{"node_id"},
	)

	// Inference metrics
	inferenceRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "distributed_llm_inference_requests_total",
			Help: "Total number of inference requests",
		},
		[]string{"node_id", "model_id", "status"},
	)

	inferenceLatencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "distributed_llm_inference_latency_seconds",
			Help:    "Inference request latency in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0},
		},
		[]string{"node_id", "model_id"},
	)

	inferenceTokensGenerated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "distributed_llm_inference_tokens_generated_total",
			Help: "Total number of tokens generated",
		},
		[]string{"node_id", "model_id"},
	)

	// Model metrics
	modelsLoadedGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "distributed_llm_models_loaded",
			Help: "Number of models loaded",
		},
		[]string{"node_id"},
	)

	modelSizeBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "distributed_llm_model_size_bytes",
			Help: "Size of loaded models in bytes",
		},
		[]string{"node_id", "model_id"},
	)

	// System metrics
	systemMemoryUsageBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "distributed_llm_system_memory_usage_bytes",
			Help: "System memory usage in bytes",
		},
		[]string{"node_id", "memory_type"},
	)

	systemCPUUsagePercent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "distributed_llm_system_cpu_usage_percent",
			Help: "System CPU usage percentage",
		},
		[]string{"node_id"},
	)

	systemGoroutinesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "distributed_llm_system_goroutines",
			Help: "Number of goroutines",
		},
		[]string{"node_id"},
	)

	// Health check metrics
	healthCheckTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "distributed_llm_health_checks_total",
			Help: "Total number of health checks",
		},
		[]string{"node_id", "status"},
	)

	healthCheckLatencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "distributed_llm_health_check_latency_seconds",
			Help:    "Health check latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		},
		[]string{"node_id"},
	)
)

// MetricsCollector manages metrics collection and export
type MetricsCollector struct {
	nodeID     string
	registry   *prometheus.Registry
	server     *http.Server
	startTime  time.Time
	logger     *slog.Logger
	cancelFunc context.CancelFunc
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(nodeID string, port int) *MetricsCollector {
	registry := prometheus.NewRegistry()

	// Register all metrics
	registry.MustRegister(
		nodeResourcesGauge,
		nodeStatusGauge,
		nodeUptimeGauge,
		layersAllocatedGauge,
		layersCapacityGauge,
		networkMessagesTotal,
		networkLatencyHistogram,
		networkConnectionsGauge,
		inferenceRequestsTotal,
		inferenceLatencyHistogram,
		inferenceTokensGenerated,
		modelsLoadedGauge,
		modelSizeBytes,
		systemMemoryUsageBytes,
		systemCPUUsagePercent,
		systemGoroutinesGauge,
		healthCheckTotal,
		healthCheckLatencyHistogram,
	)

	// Add Go runtime metrics
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return &MetricsCollector{
		nodeID:    nodeID,
		registry:  registry,
		server:    server,
		startTime: time.Now(),
		logger:    slog.With("component", "metrics"),
	}
}

// Start begins the metrics server and periodic collection
func (mc *MetricsCollector) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	mc.cancelFunc = cancel

	// Start HTTP server
	go func() {
		mc.logger.Info("Starting metrics server", "addr", mc.server.Addr)
		if err := mc.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			mc.logger.Error("Metrics server error", "error", err)
		}
	}()

	// Start periodic metrics collection
	go mc.collectSystemMetrics(ctx)

	return nil
}

// Stop shuts down the metrics collector
func (mc *MetricsCollector) Stop() error {
	if mc.cancelFunc != nil {
		mc.cancelFunc()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return mc.server.Shutdown(ctx)
}

// collectSystemMetrics periodically collects system-level metrics
func (mc *MetricsCollector) collectSystemMetrics(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.updateSystemMetrics()
		}
	}
}

// updateSystemMetrics updates system-level metrics
func (mc *MetricsCollector) updateSystemMetrics() {
	// Update uptime
	uptime := time.Since(mc.startTime).Seconds()
	nodeUptimeGauge.WithLabelValues(mc.nodeID).Set(uptime)

	// Update goroutine count
	goroutines := runtime.NumGoroutine()
	systemGoroutinesGauge.WithLabelValues(mc.nodeID).Set(float64(goroutines))

	// Update memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	systemMemoryUsageBytes.WithLabelValues(mc.nodeID, "heap_alloc").Set(float64(memStats.HeapAlloc))
	systemMemoryUsageBytes.WithLabelValues(mc.nodeID, "heap_sys").Set(float64(memStats.HeapSys))
	systemMemoryUsageBytes.WithLabelValues(mc.nodeID, "stack_sys").Set(float64(memStats.StackSys))
}

// UpdateNodeResources updates node resource metrics
func (mc *MetricsCollector) UpdateNodeResources(resources models.ResourceInfo) {
	nodeResourcesGauge.WithLabelValues(mc.nodeID, "cpu_cores").Set(float64(resources.CPUCores))
	nodeResourcesGauge.WithLabelValues(mc.nodeID, "memory_mb").Set(float64(resources.MemoryMB))
	layersCapacityGauge.WithLabelValues(mc.nodeID).Set(float64(resources.MaxLayers))
	layersAllocatedGauge.WithLabelValues(mc.nodeID, "total").Set(float64(resources.UsedLayers))

	// Update GPU metrics
	for i, gpu := range resources.GPUs {
		gpuLabel := fmt.Sprintf("gpu_%d", i)
		nodeResourcesGauge.WithLabelValues(mc.nodeID, gpuLabel+"_memory_mb").Set(float64(gpu.MemoryMB))
	}
}

// UpdateNodeStatus updates node status metrics
func (mc *MetricsCollector) UpdateNodeStatus(status models.NodeStatus) {
	var statusValue float64
	switch status {
	case models.NodeStatusOffline:
		statusValue = 0
	case models.NodeStatusOnline:
		statusValue = 1
	case models.NodeStatusBusy:
		statusValue = 2
	default:
		statusValue = 3 // Unknown/error status
	}
	nodeStatusGauge.WithLabelValues(mc.nodeID).Set(statusValue)
}

// RecordNetworkMessage records a network message
func (mc *MetricsCollector) RecordNetworkMessage(direction, messageType string) {
	networkMessagesTotal.WithLabelValues(mc.nodeID, direction, messageType).Inc()
}

// RecordNetworkLatency records network request latency
func (mc *MetricsCollector) RecordNetworkLatency(targetNode, operation string, duration time.Duration) {
	networkLatencyHistogram.WithLabelValues(mc.nodeID, targetNode, operation).Observe(duration.Seconds())
}

// UpdateNetworkConnections updates the number of active network connections
func (mc *MetricsCollector) UpdateNetworkConnections(count int) {
	networkConnectionsGauge.WithLabelValues(mc.nodeID).Set(float64(count))
}

// RecordInferenceRequest records an inference request
func (mc *MetricsCollector) RecordInferenceRequest(modelID, status string, duration time.Duration, tokensGenerated int) {
	inferenceRequestsTotal.WithLabelValues(mc.nodeID, modelID, status).Inc()
	inferenceLatencyHistogram.WithLabelValues(mc.nodeID, modelID).Observe(duration.Seconds())
	if tokensGenerated > 0 {
		inferenceTokensGenerated.WithLabelValues(mc.nodeID, modelID).Add(float64(tokensGenerated))
	}
}

// UpdateModelsLoaded updates the number of loaded models
func (mc *MetricsCollector) UpdateModelsLoaded(count int) {
	modelsLoadedGauge.WithLabelValues(mc.nodeID).Set(float64(count))
}

// UpdateModelSize updates the size of a loaded model
func (mc *MetricsCollector) UpdateModelSize(modelID string, sizeBytes int64) {
	modelSizeBytes.WithLabelValues(mc.nodeID, modelID).Set(float64(sizeBytes))
}

// RecordHealthCheck records a health check
func (mc *MetricsCollector) RecordHealthCheck(status string, duration time.Duration) {
	healthCheckTotal.WithLabelValues(mc.nodeID, status).Inc()
	healthCheckLatencyHistogram.WithLabelValues(mc.nodeID).Observe(duration.Seconds())
}

// UpdateLayerAllocation updates layer allocation for a specific model
func (mc *MetricsCollector) UpdateLayerAllocation(modelID string, allocatedLayers int) {
	layersAllocatedGauge.WithLabelValues(mc.nodeID, modelID).Set(float64(allocatedLayers))
}
