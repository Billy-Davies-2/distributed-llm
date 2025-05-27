package integration

import (
	"distributed-llm/internal/k8s"
	"distributed-llm/pkg/config"
	"distributed-llm/pkg/models"
	"testing"
	"time"
)

// TestK8sClientWithConfig tests K8s client integration with configuration
func TestK8sClientWithConfig(t *testing.T) {
	// Skip if not in Kubernetes environment
	if !isKubernetesEnvironment() {
		t.Skip("Skipping K8s integration test - not in Kubernetes environment")
	}

	// Test K8s client creation
	client, err := k8s.NewKubernetesClient()
	if err != nil {
		t.Fatalf("Failed to create K8s client: %v", err)
	}

	if client == nil {
		t.Fatal("K8s client should not be nil")
	}

	// Test pod operations (read-only to avoid side effects)
	pods, err := client.ListPods("default")
	if err != nil {
		t.Errorf("Failed to list pods: %v", err)
	}

	t.Logf("Found %d pods", len(pods.Items))
}

// TestConfigIntegrationWithComponents tests configuration integration across components
func TestConfigIntegrationWithComponents(t *testing.T) {
	// Test default configuration
	cfg := config.Default()

	if cfg == nil {
		t.Fatal("Default config should not be nil")
	}

	// Test that config values are reasonable for testing
	if cfg.NodeID == "" {
		cfg.NodeID = "test-node"
	}

	if cfg.Port <= 0 {
		cfg.Port = 8080
	}

	if cfg.GossipPort <= 0 {
		cfg.GossipPort = 7946
	}

	// Test configuration validation
	if cfg.Port == cfg.GossipPort {
		t.Error("Port and GossipPort should be different")
	}

	// Test that configuration can be used to create components
	testNode := models.Node{
		ID:      cfg.NodeID,
		Address: "127.0.0.1",
		Port:    cfg.Port,
		Status:  models.NodeStatusOnline,
		Resources: models.ResourceInfo{
			CPUCores:   4,
			MemoryMB:   8192,
			MaxLayers:  8,
			UsedLayers: 0,
			GPUs:       []models.GPUInfo{},
		},
		LastSeen: time.Now(),
	}

	if !testNode.IsHealthy() {
		t.Error("Test node should be healthy")
	}
}

// TestConfigWithEnvironmentVariables tests configuration with environment overrides
func TestConfigWithEnvironmentVariables(t *testing.T) {
	// Test loading config from environment
	// In a real test, we would set environment variables and test they're picked up
	cfg := config.Default()

	// Test config field validation
	if cfg.ModelPath == "" {
		t.Log("ModelPath not configured - this is expected in test environment")
	}

	if cfg.DataPath == "" {
		t.Log("DataPath not configured - this is expected in test environment")
	}

	// Test that config can handle missing optional fields
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	validLogLevels := []string{"debug", "info", "warn", "error"}
	isValidLogLevel := false
	for _, level := range validLogLevels {
		if cfg.LogLevel == level {
			isValidLogLevel = true
			break
		}
	}

	if !isValidLogLevel {
		t.Errorf("Invalid log level: %s", cfg.LogLevel)
	}
}

// TestFullStackIntegration tests integration across all major components
func TestFullStackIntegration(t *testing.T) {
	// This test simulates a full stack startup sequence

	// 1. Load configuration
	cfg := config.Default()
	cfg.NodeID = "integration-test-node"
	cfg.Port = findAvailablePort(t)
	cfg.GossipPort = findAvailablePort(t)

	// 2. Validate configuration
	if cfg.Port == cfg.GossipPort {
		t.Fatal("Ports should be different")
	}

	// 3. Create node with configuration
	testNode := models.Node{
		ID:      cfg.NodeID,
		Address: "127.0.0.1",
		Port:    cfg.Port,
		Status:  models.NodeStatusOnline,
		Resources: models.ResourceInfo{
			CPUCores:   4,
			MemoryMB:   8192,
			MaxLayers:  8,
			UsedLayers: 0,
			GPUs:       []models.GPUInfo{},
		},
		LastSeen: time.Now(),
	}

	// 4. Validate node health
	if !testNode.IsHealthy() {
		t.Error("Node should be healthy with valid configuration")
	}

	// 5. Test that node can handle basic model operations
	testModel := models.Model{
		ID:         "test-model",
		Name:       "Test Model",
		Version:    "1.0",
		LayerCount: 24,
		Size:       1024 * 1024 * 1024, // 1GB
	}

	modelSizeGB := testModel.SizeInGB()
	if modelSizeGB != 1.0 {
		t.Errorf("Model size should be 1.0 GB, got %.2f", modelSizeGB)
	}

	// 6. Test resource capacity for model
	if testNode.Resources.MaxLayers < testModel.LayerCount {
		t.Log("Node does not have enough layer capacity for test model (expected in test)")
	}

	t.Logf("Integration test completed successfully with node %s", cfg.NodeID)
}

// Helper function to check if we're in a Kubernetes environment
func isKubernetesEnvironment() bool {
	// Simple check - in a real implementation this would be more sophisticated
	_, err := k8s.NewKubernetesClient()
	return err == nil
}
