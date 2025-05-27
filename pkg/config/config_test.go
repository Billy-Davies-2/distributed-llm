package config

import (
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg == nil {
		t.Fatal("NewConfig() returned nil")
	}

	// Test default values
	if cfg.NodeID == "" {
		t.Error("Expected non-empty NodeID")
	}

	if cfg.BindPort != 8080 {
		t.Errorf("Expected default BindPort to be 8080, got %d", cfg.BindPort)
	}

	if cfg.GossipPort != 7946 {
		t.Errorf("Expected default GossipPort to be 7946, got %d", cfg.GossipPort)
	}

	if cfg.MetricsPort != 9090 {
		t.Errorf("Expected default MetricsPort to be 9090, got %d", cfg.MetricsPort)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected default LogLevel to be 'info', got %s", cfg.LogLevel)
	}

	if cfg.HealthCheckInterval != 30*time.Second {
		t.Errorf("Expected default HealthCheckInterval to be 30s, got %v", cfg.HealthCheckInterval)
	}

	if cfg.EnableCompression != true {
		t.Error("Expected default EnableCompression to be true")
	}

	if cfg.MaxConnections != 100 {
		t.Errorf("Expected default MaxConnections to be 100, got %d", cfg.MaxConnections)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  NewConfig(),
			wantErr: false,
		},
		{
			name: "empty node ID",
			config: &Config{
				NodeID:              "",
				BindPort:            8080,
				GossipPort:          7946,
				MetricsPort:         9090,
				LogLevel:            "info",
				HealthCheckInterval: 30 * time.Second,
				EnableCompression:   true,
				MaxConnections:      100,
			},
			wantErr: true,
		},
		{
			name: "invalid bind port",
			config: &Config{
				NodeID:              "test-node",
				BindPort:            -1,
				GossipPort:          7946,
				MetricsPort:         9090,
				LogLevel:            "info",
				HealthCheckInterval: 30 * time.Second,
				EnableCompression:   true,
				MaxConnections:      100,
			},
			wantErr: true,
		},
		{
			name: "port conflict",
			config: &Config{
				NodeID:              "test-node",
				BindPort:            8080,
				GossipPort:          8080, // Same as bind port
				MetricsPort:         9090,
				LogLevel:            "info",
				HealthCheckInterval: 30 * time.Second,
				EnableCompression:   true,
				MaxConnections:      100,
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				NodeID:              "test-node",
				BindPort:            8080,
				GossipPort:          7946,
				MetricsPort:         9090,
				LogLevel:            "invalid",
				HealthCheckInterval: 30 * time.Second,
				EnableCompression:   true,
				MaxConnections:      100,
			},
			wantErr: true,
		},
		{
			name: "zero health check interval",
			config: &Config{
				NodeID:              "test-node",
				BindPort:            8080,
				GossipPort:          7946,
				MetricsPort:         9090,
				LogLevel:            "info",
				HealthCheckInterval: 0,
				EnableCompression:   true,
				MaxConnections:      100,
			},
			wantErr: true,
		},
		{
			name: "invalid max connections",
			config: &Config{
				NodeID:              "test-node",
				BindPort:            8080,
				GossipPort:          7946,
				MetricsPort:         9090,
				LogLevel:            "info",
				HealthCheckInterval: 30 * time.Second,
				EnableCompression:   true,
				MaxConnections:      0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_GetAddress(t *testing.T) {
	cfg := NewConfig()
	cfg.BindAddress = "localhost"
	cfg.BindPort = 8080

	expected := "localhost:8080"
	actual := cfg.GetAddress()

	if actual != expected {
		t.Errorf("Expected address %s, got %s", expected, actual)
	}
}

func TestConfig_GetGossipAddress(t *testing.T) {
	cfg := NewConfig()
	cfg.BindAddress = "localhost"
	cfg.GossipPort = 7946

	expected := "localhost:7946"
	actual := cfg.GetGossipAddress()

	if actual != expected {
		t.Errorf("Expected gossip address %s, got %s", expected, actual)
	}
}

func TestConfig_GetMetricsAddress(t *testing.T) {
	cfg := NewConfig()
	cfg.BindAddress = "localhost"
	cfg.MetricsPort = 9090

	expected := "localhost:9090"
	actual := cfg.GetMetricsAddress()

	if actual != expected {
		t.Errorf("Expected metrics address %s, got %s", expected, actual)
	}
}

func TestConfig_IsGPUEnabled(t *testing.T) {
	cfg := NewConfig()

	// Test default value
	if cfg.IsGPUEnabled() {
		t.Error("Expected GPU to be disabled by default")
	}

	// Test with GPU enabled
	cfg.EnableGPU = true
	if !cfg.IsGPUEnabled() {
		t.Error("Expected GPU to be enabled")
	}
}

func TestConfig_GetSeedNodes(t *testing.T) {
	cfg := NewConfig()

	// Test empty seed nodes
	seeds := cfg.GetSeedNodes()
	if len(seeds) != 0 {
		t.Errorf("Expected empty seed nodes, got %v", seeds)
	}

	// Test with seed nodes
	cfg.SeedNodes = []string{"localhost:8080", "localhost:8081"}
	seeds = cfg.GetSeedNodes()

	if len(seeds) != 2 {
		t.Errorf("Expected 2 seed nodes, got %d", len(seeds))
	}

	expected := []string{"localhost:8080", "localhost:8081"}
	for i, seed := range seeds {
		if seed != expected[i] {
			t.Errorf("Expected seed node %s at index %d, got %s", expected[i], i, seed)
		}
	}
}
