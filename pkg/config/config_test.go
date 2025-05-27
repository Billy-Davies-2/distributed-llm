package config

import (
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg == nil {
		t.Fatal("Default() returned nil")
	}

	// Test default values
	if cfg.NodeID != "default-node" {
		t.Errorf("Expected NodeID to be 'default-node', got %s", cfg.NodeID)
	}

	if cfg.Port != 8080 {
		t.Errorf("Expected default Port to be 8080, got %d", cfg.Port)
	}

	if cfg.GossipPort != 7946 {
		t.Errorf("Expected default GossipPort to be 7946, got %d", cfg.GossipPort)
	}

	if cfg.KubernetesNamespace != "default" {
		t.Errorf("Expected default KubernetesNamespace to be 'default', got %s", cfg.KubernetesNamespace)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected default LogLevel to be 'info', got %s", cfg.LogLevel)
	}

	if cfg.ModelPath != "/models" {
		t.Errorf("Expected default ModelPath to be '/models', got %s", cfg.ModelPath)
	}

	if cfg.DataPath != "/data" {
		t.Errorf("Expected default DataPath to be '/data', got %s", cfg.DataPath)
	}

	// Test resource limits
	if cfg.ResourceLimits.CPU != "1000m" {
		t.Errorf("Expected default CPU limit to be '1000m', got %s", cfg.ResourceLimits.CPU)
	}

	if cfg.ResourceLimits.Memory != "2Gi" {
		t.Errorf("Expected default Memory limit to be '2Gi', got %s", cfg.ResourceLimits.Memory)
	}

	if cfg.ResourceLimits.GPU != "1" {
		t.Errorf("Expected default GPU limit to be '1', got %s", cfg.ResourceLimits.GPU)
	}
}

func TestLoadConfig(t *testing.T) {
	// Test loading non-existent file
	_, err := LoadConfig("non-existent-file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}

	// Note: Testing with actual file would require creating a temp file
	// For now, we test the error case
}

func TestResourceLimits(t *testing.T) {
	limits := ResourceLimits{
		CPU:    "2000m",
		Memory: "4Gi",
		GPU:    "2",
	}

	if limits.CPU != "2000m" {
		t.Errorf("Expected CPU '2000m', got %s", limits.CPU)
	}

	if limits.Memory != "4Gi" {
		t.Errorf("Expected Memory '4Gi', got %s", limits.Memory)
	}

	if limits.GPU != "2" {
		t.Errorf("Expected GPU '2', got %s", limits.GPU)
	}
}
