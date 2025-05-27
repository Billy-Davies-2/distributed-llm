package fuzz

import (
	"distributed-llm/internal/network"
	"distributed-llm/pkg/models"
	"strings"
	"testing"
	"time"
)

// FuzzP2PNetworkCreation fuzzes P2P network creation with random inputs
func FuzzP2PNetworkCreation(f *testing.F) {
	// Seed the fuzzer with some basic test cases
	f.Add("node-1", 8080, 7946)
	f.Add("", 0, 0)
	f.Add("very-long-node-name-that-might-cause-issues", 65536, 65537)
	f.Add("node with spaces", -1, -2)
	f.Add("node\nwith\nnewlines", 1024, 2048)
	f.Add("节点中文", 8000, 9000)

	f.Fuzz(func(t *testing.T, nodeID string, bindPort, gossipPort int) {
		// Test that the function doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("P2P network creation panicked with nodeID=%q, bindPort=%d, gossipPort=%d: %v",
					nodeID, bindPort, gossipPort, r)
			}
		}()

		network, err := network.NewP2PNetwork(nodeID, bindPort, gossipPort)

		// Valid inputs should not return an error
		if isValidInput(nodeID, bindPort, gossipPort) {
			if err != nil {
				t.Errorf("Valid input should not produce error: nodeID=%q, bindPort=%d, gossipPort=%d, error=%v",
					nodeID, bindPort, gossipPort, err)
			}
			if network == nil {
				t.Errorf("Valid input should produce non-nil network")
			}
		} else {
			// Invalid inputs should return an error
			if err == nil {
				t.Errorf("Invalid input should produce error: nodeID=%q, bindPort=%d, gossipPort=%d",
					nodeID, bindPort, gossipPort)
			}
		}
	})
}

// FuzzNodeValidation fuzzes node validation with random data
func FuzzNodeValidation(f *testing.F) {
	// Seed with test cases
	f.Add("node-1", "127.0.0.1", 8080, int64(4), int64(8192), int32(8), int32(0))
	f.Add("", "", 0, int64(0), int64(0), int32(0), int32(0))
	f.Add("node", "invalid-ip", -1, int64(-1), int64(-1), int32(-1), int32(-1))

	f.Fuzz(func(t *testing.T, id, address string, port int, cpuCores, memoryMB int64, maxLayers, usedLayers int32) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Node validation panicked: %v", r)
			}
		}()

		node := models.Node{
			ID:       id,
			Address:  address,
			Port:     port,
			Status:   models.NodeStatusOnline,
			LastSeen: time.Now(), // Set recent time for health check
			Resources: models.ResourceInfo{
				CPUCores:   cpuCores,
				MemoryMB:   memoryMB,
				MaxLayers:  maxLayers,
				UsedLayers: usedLayers,
				GPUs:       []models.GPUInfo{},
			},
		}

		// Test IsHealthy doesn't panic
		healthy := node.IsHealthy()

		// Basic validation logic
		expectedHealthy := isValidNode(id, address, port, cpuCores, memoryMB, maxLayers, usedLayers)

		if healthy != expectedHealthy {
			t.Logf("Health mismatch for node: %+v, expected=%v, got=%v", node, expectedHealthy, healthy)
		}
	})
}

// FuzzModelValidation fuzzes model validation
func FuzzModelValidation(f *testing.F) {
	// Seed with test cases
	f.Add("model-1", "Test Model", "1.0", int32(24), int64(1024*1024*1024))
	f.Add("", "", "", int32(0), int64(0))
	f.Add("model", "name", "version", int32(-1), int64(-1))

	f.Fuzz(func(t *testing.T, id, name, version string, layerCount int32, size int64) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Model validation panicked: %v", r)
			}
		}()

		model := models.Model{
			ID:         id,
			Name:       name,
			Version:    version,
			LayerCount: layerCount,
			Size:       size,
		}

		// Test SizeInGB doesn't panic and returns reasonable values
		sizeGB := model.SizeInGB()

		if size > 0 {
			expectedSizeGB := float64(size) / (1024 * 1024 * 1024)
			tolerance := 0.01
			if abs(sizeGB-expectedSizeGB) > tolerance {
				t.Errorf("Size calculation incorrect: size=%d, expected=%.2f GB, got=%.2f GB",
					size, expectedSizeGB, sizeGB)
			}
		}

		// Negative sizes should result in 0 or handle gracefully
		if size < 0 && sizeGB > 0 {
			t.Errorf("Negative size should not result in positive GB value: size=%d, sizeGB=%.2f",
				size, sizeGB)
		}
	})
}

// FuzzResourceInfoValidation fuzzes resource info validation
func FuzzResourceInfoValidation(f *testing.F) {
	// Seed with test cases
	f.Add(int64(4), int64(8192), int32(8), int32(4))
	f.Add(int64(0), int64(0), int32(0), int32(0))
	f.Add(int64(-1), int64(-1), int32(-1), int32(-1))
	f.Add(int64(1000), int64(1000000), int32(100), int32(200)) // used > max

	f.Fuzz(func(t *testing.T, cpuCores, memoryMB int64, maxLayers, usedLayers int32) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ResourceInfo validation panicked: %v", r)
			}
		}()

		resources := models.ResourceInfo{
			CPUCores:   cpuCores,
			MemoryMB:   memoryMB,
			MaxLayers:  maxLayers,
			UsedLayers: usedLayers,
			GPUs:       []models.GPUInfo{},
		}

		// Test TotalMemoryMB doesn't panic
		totalMem := resources.TotalMemoryMB()

		// Should equal MemoryMB since no GPUs
		if totalMem != memoryMB {
			t.Errorf("TotalMemoryMB should equal MemoryMB when no GPUs: expected=%d, got=%d",
				memoryMB, totalMem)
		}

		// Test HasGPUs doesn't panic
		hasGPUs := resources.HasGPUs()
		if hasGPUs {
			t.Error("Should not have GPUs in this test")
		}

		// Validate resource constraints
		if usedLayers > maxLayers && maxLayers >= 0 {
			t.Logf("Used layers (%d) exceeds max layers (%d) - this should be handled gracefully",
				usedLayers, maxLayers)
		}
	})
}

// FuzzGPUInfoValidation fuzzes GPU info validation
func FuzzGPUInfoValidation(f *testing.F) {
	// Seed with test cases
	f.Add("NVIDIA RTX 4090", int64(24576), "GPU-12345")
	f.Add("", int64(0), "")
	f.Add("Very Long GPU Name That Might Cause Buffer Issues", int64(-1), "invalid-uuid")

	f.Fuzz(func(t *testing.T, name string, memoryMB int64, uuid string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("GPUInfo validation panicked: %v", r)
			}
		}()

		gpu := models.GPUInfo{
			Name:     name,
			MemoryMB: memoryMB,
			UUID:     uuid,
		}

		// Test in ResourceInfo context
		resources := models.ResourceInfo{
			CPUCores:   4,
			MemoryMB:   8192,
			MaxLayers:  8,
			UsedLayers: 0,
			GPUs:       []models.GPUInfo{gpu},
		}

		// Test methods don't panic
		hasGPUs := resources.HasGPUs()
		totalMem := resources.TotalMemoryMB()

		// Should have GPUs
		if !hasGPUs {
			t.Error("Should have GPUs when GPU slice is non-empty")
		}

		// Total memory should include GPU memory
		expectedTotal := int64(8192) + memoryMB
		if memoryMB >= 0 && totalMem != expectedTotal {
			t.Errorf("Total memory incorrect: expected=%d, got=%d", expectedTotal, totalMem)
		}
	})
}

// Helper functions for validation

func isValidInput(nodeID string, bindPort, gossipPort int) bool {
	// Basic validation rules
	if nodeID == "" {
		return false
	}
	if bindPort <= 0 || bindPort > 65535 {
		return false
	}
	if gossipPort <= 0 || gossipPort > 65535 {
		return false
	}
	if bindPort == gossipPort {
		return false
	}
	if strings.Contains(nodeID, "\n") || strings.Contains(nodeID, "\r") {
		return false
	}
	return true
}

func isValidNode(id, address string, port int, cpuCores, memoryMB int64, maxLayers, usedLayers int32) bool {
	if id == "" || address == "" {
		return false
	}
	if port <= 0 || port > 65535 {
		return false
	}
	if cpuCores <= 0 || memoryMB <= 0 {
		return false
	}
	if maxLayers < 0 || usedLayers < 0 {
		return false
	}
	if usedLayers > maxLayers {
		return false
	}
	return true
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
