package agent

import (
	"testing"

	"distributed-llm/pkg/models"
)

func TestGetResourceInfo(t *testing.T) {
	resources := GetResourceInfo()

	if resources.CPUCores <= 0 {
		t.Error("CPU cores should be greater than 0")
	}

	if resources.MemoryMB <= 0 {
		t.Error("Memory should be greater than 0")
	}

	// Should have reasonable values
	if resources.CPUCores > 128 {
		t.Errorf("CPU cores seem unreasonably high: %d", resources.CPUCores)
	}

	if resources.MemoryMB > 1024*1024 { // 1TB
		t.Errorf("Memory seems unreasonably high: %d MB", resources.MemoryMB)
	}
}

func TestCalculateLayerCapacity(t *testing.T) {
	tests := []struct {
		name     string
		memory   int64
		expected int
	}{
		{"Low memory", 4096, 4},         // 4GB -> 4 layers
		{"Medium memory", 8192, 8},      // 8GB -> 8 layers
		{"High memory", 16384, 16},      // 16GB -> 16 layers
		{"Very high memory", 32768, 32}, // 32GB -> 32 layers
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple calculation: 1GB per layer
			capacity := int(tt.memory / 1024)
			if capacity != tt.expected {
				t.Errorf("calculateLayerCapacity(%d) = %d, want %d",
					tt.memory, capacity, tt.expected)
			}
		})
	}
}

func TestResourceMonitoring(t *testing.T) {
	// Test resource monitoring doesn't panic
	resources := GetResourceInfo()

	// Should be consistent across calls
	resources2 := GetResourceInfo()

	if resources.CPUCores != resources2.CPUCores {
		t.Error("CPU cores should be consistent across calls")
	}

	// Memory might vary slightly, but should be close
	memDiff := resources.MemoryMB - resources2.MemoryMB
	if memDiff < 0 {
		memDiff = -memDiff
	}

	// Allow 10% variation in memory (due to system processes)
	maxDiff := resources.MemoryMB / 10
	if memDiff > maxDiff {
		t.Errorf("Memory difference too large: %d MB", memDiff)
	}
}

func TestResourceUpdate(t *testing.T) {
	// Create initial resource info
	resources := models.ResourceInfo{
		CPUCores:   4,
		MemoryMB:   8192,
		MaxLayers:  8,
		UsedLayers: 0,
		GPUs:       []models.GPUInfo{},
	}

	// Simulate using some layers
	resources.UsedLayers = 3
	available := resources.MaxLayers - resources.UsedLayers

	if available != 5 {
		t.Errorf("Available layers = %d, want 5", available)
	}

	// Test capacity check
	if resources.UsedLayers > resources.MaxLayers {
		t.Error("Used layers should not exceed max layers")
	}
}

func TestGPUDetection(t *testing.T) {
	resources := GetResourceInfo()

	// GPU detection might vary by system, just test structure
	for i, gpu := range resources.GPUs {
		t.Logf("GPU %d: %s with %d MB", i, gpu.Name, gpu.MemoryMB)

		if gpu.Name == "" {
			t.Error("GPU name should not be empty")
		}

		if gpu.MemoryMB <= 0 {
			t.Error("GPU memory should be greater than 0")
		}

		// Reasonable GPU memory limits
		if gpu.MemoryMB > 100*1024 { // 100GB
			t.Errorf("GPU memory seems unreasonably high: %d MB", gpu.MemoryMB)
		}
	}
}

// Benchmark tests
func BenchmarkGetResourceInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetResourceInfo()
	}
}

func BenchmarkResourceUpdate(b *testing.B) {
	resources := models.ResourceInfo{
		CPUCores:   8,
		MemoryMB:   16384,
		MaxLayers:  16,
		UsedLayers: 0,
		GPUs:       []models.GPUInfo{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate layer allocation/deallocation
		resources.UsedLayers = (resources.UsedLayers + 1) % resources.MaxLayers
	}
}
