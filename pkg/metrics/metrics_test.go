package metrics

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"distributed-llm/pkg/models"
)

func TestNewMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector("test-node", 9090)
	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	if collector.nodeID != "test-node" {
		t.Errorf("Expected nodeID 'test-node', got '%s'", collector.nodeID)
	}

	if collector.registry == nil {
		t.Error("Expected non-nil registry")
	}
}

func TestMetricsCollectorStartStop(t *testing.T) {
	collector := NewMetricsCollector("test-node", 9091)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test that metrics endpoint is accessible
	resp, err := http.Get("http://localhost:9091/metrics")
	if err != nil {
		t.Errorf("Failed to get metrics: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	}

	// Test health endpoint
	resp, err = http.Get("http://localhost:9091/health")
	if err != nil {
		t.Errorf("Failed to get health: %v", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("Failed to read response body: %v", err)
		} else if string(body) != "OK" {
			t.Errorf("Expected 'OK', got '%s'", string(body))
		}
	}

	err = collector.Stop()
	if err != nil {
		t.Errorf("Failed to stop collector: %v", err)
	}
}

func TestUpdateNodeResources(t *testing.T) {
	collector := NewMetricsCollector("test-node", 9092)

	resources := models.ResourceInfo{
		CPUCores:   8,
		MemoryMB:   16384,
		GPUs:       []models.GPUInfo{{Name: "RTX 4090", MemoryMB: 24576, UUID: "test-uuid"}},
		MaxLayers:  32,
		UsedLayers: 16,
	}

	// This should not panic
	collector.UpdateNodeResources(resources)
}

func TestUpdateNodeStatus(t *testing.T) {
	collector := NewMetricsCollector("test-node", 9093)

	// Test all valid statuses
	statuses := []models.NodeStatus{
		models.NodeStatusOnline,
		models.NodeStatusOffline,
		models.NodeStatusBusy,
	}

	for _, status := range statuses {
		collector.UpdateNodeStatus(status)
	}
}

func TestRecordInferenceRequest(t *testing.T) {
	collector := NewMetricsCollector("test-node", 9094)

	duration := 1500 * time.Millisecond
	collector.RecordInferenceRequest("llama-7b", "success", duration, 150)
	collector.RecordInferenceRequest("llama-7b", "error", duration, 0)
}

func TestNetworkMetrics(t *testing.T) {
	collector := NewMetricsCollector("test-node", 9095)

	collector.RecordNetworkMessage("outbound", "register_node")
	collector.RecordNetworkMessage("inbound", "health_check")

	duration := 50 * time.Millisecond
	collector.RecordNetworkLatency("node-2", "health_check", duration)

	collector.UpdateNetworkConnections(5)
}

func TestModelMetrics(t *testing.T) {
	collector := NewMetricsCollector("test-node", 9096)

	collector.UpdateModelsLoaded(3)
	collector.UpdateModelSize("llama-7b", 7000000000) // 7GB
	collector.UpdateLayerAllocation("llama-7b", 16)
}

func TestHealthCheckMetrics(t *testing.T) {
	collector := NewMetricsCollector("test-node", 9097)

	duration := 5 * time.Millisecond
	collector.RecordHealthCheck("success", duration)
	collector.RecordHealthCheck("failure", duration)
}

func TestSystemMetricsCollection(t *testing.T) {
	collector := NewMetricsCollector("test-node", 9098)

	// Test that updateSystemMetrics doesn't panic
	collector.updateSystemMetrics()

	// Start collector and let it run briefly to test periodic collection
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Wait for at least one collection cycle
	time.Sleep(100 * time.Millisecond)

	collector.Stop()
}

// Benchmark tests for performance
func BenchmarkUpdateNodeResources(b *testing.B) {
	collector := NewMetricsCollector("test-node", 9099)

	resources := models.ResourceInfo{
		CPUCores:   8,
		MemoryMB:   16384,
		GPUs:       []models.GPUInfo{{Name: "RTX 4090", MemoryMB: 24576, UUID: "test-uuid"}},
		MaxLayers:  32,
		UsedLayers: 16,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.UpdateNodeResources(resources)
	}
}

func BenchmarkRecordNetworkMessage(b *testing.B) {
	collector := NewMetricsCollector("test-node", 9100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordNetworkMessage("outbound", "register_node")
	}
}

func BenchmarkRecordInferenceRequest(b *testing.B) {
	collector := NewMetricsCollector("test-node", 9101)

	duration := 1500 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordInferenceRequest("llama-7b", "success", duration, 150)
	}
}
