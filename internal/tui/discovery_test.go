package tui

import (
	"fmt"
	"testing"
	"time"

	"distributed-llm/pkg/models"
)

func TestNewAgentDiscovery(t *testing.T) {
	updateChan := make(chan []models.Node, 10)

	tests := []struct {
		name   string
		config DiscoveryConfig
	}{
		{
			name: "basic discovery config",
			config: DiscoveryConfig{
				UpdateChan: updateChan,
			},
		},
		{
			name: "discovery with seed nodes",
			config: DiscoveryConfig{
				UpdateChan: updateChan,
				SeedNodes:  []string{"localhost:8080", "localhost:8081"},
			},
		},
		{
			name: "discovery with docker mode",
			config: DiscoveryConfig{
				UpdateChan: updateChan,
				DockerMode: true,
			},
		},
		{
			name: "discovery with k8s namespace",
			config: DiscoveryConfig{
				UpdateChan:   updateChan,
				K8sNamespace: "distributed-llm",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discovery := NewAgentDiscovery(tt.config)

			if discovery == nil {
				t.Fatal("NewAgentDiscovery returned nil")
			}

			if discovery.logger == nil {
				t.Error("Logger not set")
			}

			if discovery.updateChan != tt.config.UpdateChan {
				t.Error("UpdateChan not set correctly")
			}

			if len(tt.config.SeedNodes) > 0 && len(discovery.seedNodes) != len(tt.config.SeedNodes) {
				t.Errorf("Expected %d seed nodes, got %d", len(tt.config.SeedNodes), len(discovery.seedNodes))
			}

			if discovery.dockerMode != tt.config.DockerMode {
				t.Errorf("Expected docker mode %v, got %v", tt.config.DockerMode, discovery.dockerMode)
			}

			if discovery.k8sNamespace != tt.config.K8sNamespace {
				t.Errorf("Expected k8s namespace %s, got %s", tt.config.K8sNamespace, discovery.k8sNamespace)
			}
		})
	}
}

func TestAgentDiscovery_Start(t *testing.T) {
	updateChan := make(chan []models.Node, 10)

	config := DiscoveryConfig{
		UpdateChan: updateChan,
	}

	discovery := NewAgentDiscovery(config)

	if discovery == nil {
		t.Fatal("NewAgentDiscovery returned nil")
	}

	// Test starting discovery (this will run briefly and exit)
	done := make(chan error, 1)
	go func() {
		err := discovery.Start()
		done <- err
	}()

	// Allow some time for the discovery to start
	time.Sleep(100 * time.Millisecond)

	// Test stopping discovery
	discovery.Stop()

	// Wait for start to complete
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start() returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Start() did not complete in time")
	}
}

func TestAgentDiscovery_Stop(t *testing.T) {
	updateChan := make(chan []models.Node, 10)

	config := DiscoveryConfig{
		UpdateChan: updateChan,
	}

	discovery := NewAgentDiscovery(config)

	if discovery == nil {
		t.Fatal("NewAgentDiscovery returned nil")
	}

	// Test that Stop doesn't panic when called before Start
	discovery.Stop()

	// Test that Stop can be called multiple times
	discovery.Stop()
}

func TestAgentDiscovery_GetNodes(t *testing.T) {
	updateChan := make(chan []models.Node, 10)

	config := DiscoveryConfig{
		UpdateChan: updateChan,
	}

	discovery := NewAgentDiscovery(config)

	if discovery == nil {
		t.Fatal("NewAgentDiscovery returned nil")
	}

	// Initially should have no nodes
	nodes := discovery.GetNodes()
	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes initially, got %d", len(nodes))
	}

	// Add a test node manually for coverage
	testNode := &models.Node{
		ID:      "test-node-1",
		Address: "localhost",
		Port:    8080,
		Status:  models.NodeStatusOnline,
	}

	discovery.mu.Lock()
	discovery.nodes["test-node-1"] = testNode
	discovery.mu.Unlock()

	nodes = discovery.GetNodes()
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node after adding, got %d", len(nodes))
	}

	if nodes[0].ID != "test-node-1" {
		t.Errorf("Expected node ID 'test-node-1', got '%s'", nodes[0].ID)
	}
}

func TestAgentDiscovery_GetClient(t *testing.T) {
	updateChan := make(chan []models.Node, 10)

	config := DiscoveryConfig{
		UpdateChan: updateChan,
	}

	discovery := NewAgentDiscovery(config)

	if discovery == nil {
		t.Fatal("NewAgentDiscovery returned nil")
	}

	// Test getting client for non-existent node
	client, exists := discovery.GetClient("non-existent")
	if client != nil {
		t.Error("Expected nil client for non-existent node")
	}
	if exists {
		t.Error("Expected exists to be false for non-existent node")
	}

	// Add a test client manually for coverage
	testClient := NewClient("localhost:8080")
	discovery.mu.Lock()
	discovery.clients["test-node"] = testClient
	discovery.mu.Unlock()

	client, exists = discovery.GetClient("test-node")
	if client == nil {
		t.Error("Expected client for existing node")
	}
	if !exists {
		t.Error("Expected exists to be true for existing node")
	}

	if client != testClient {
		t.Error("Expected to get the same client instance")
	}
}

func TestAgentDiscovery_EdgeCases(t *testing.T) {
	t.Run("nil update channel", func(t *testing.T) {
		config := DiscoveryConfig{
			UpdateChan: nil,
		}

		discovery := NewAgentDiscovery(config)
		if discovery == nil {
			t.Fatal("NewAgentDiscovery returned nil")
		}

		// Should not panic with nil channel
		done := make(chan error, 1)
		go func() {
			err := discovery.Start()
			done <- err
		}()

		time.Sleep(50 * time.Millisecond)
		discovery.Stop()

		select {
		case <-done:
			// Success
		case <-time.After(500 * time.Millisecond):
			t.Error("Start() did not complete with nil channel")
		}
	})

	t.Run("empty seed nodes", func(t *testing.T) {
		updateChan := make(chan []models.Node, 10)
		config := DiscoveryConfig{
			UpdateChan: updateChan,
			SeedNodes:  []string{},
		}

		discovery := NewAgentDiscovery(config)
		if discovery == nil {
			t.Fatal("NewAgentDiscovery returned nil")
		}

		// Should handle empty seed nodes gracefully
		nodes := discovery.GetNodes()
		if nodes == nil {
			t.Error("GetNodes should not return nil")
		}
	})

	t.Run("invalid seed nodes", func(t *testing.T) {
		updateChan := make(chan []models.Node, 10)
		config := DiscoveryConfig{
			UpdateChan: updateChan,
			SeedNodes:  []string{"invalid-address", "", "localhost:99999"},
		}

		discovery := NewAgentDiscovery(config)
		if discovery == nil {
			t.Fatal("NewAgentDiscovery returned nil")
		}

		// Should handle invalid seed nodes gracefully
		done := make(chan error, 1)
		go func() {
			err := discovery.Start()
			done <- err
		}()

		time.Sleep(100 * time.Millisecond)
		discovery.Stop()

		// Wait for completion or timeout
		select {
		case <-done:
			// Success - function completed despite invalid addresses
		case <-time.After(300 * time.Millisecond):
			t.Error("Start() took too long with invalid addresses")
		}
	})
}

func TestAgentDiscovery_ConcurrentOperations(t *testing.T) {
	updateChan := make(chan []models.Node, 10)

	config := DiscoveryConfig{
		UpdateChan: updateChan,
		SeedNodes:  []string{"localhost:8080"},
	}

	discovery := NewAgentDiscovery(config)

	if discovery == nil {
		t.Fatal("NewAgentDiscovery returned nil")
	}

	// Test concurrent start/stop operations
	done := make(chan bool, 2)

	go func() {
		discovery.Start()
		done <- true
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		discovery.Stop()
		done <- true
	}()

	// Wait for both operations to complete
	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(500 * time.Millisecond):
			t.Error("Concurrent operations took too long")
			return
		}
	}

	// Test concurrent GetNodes calls
	for i := 0; i < 10; i++ {
		go func() {
			nodes := discovery.GetNodes()
			if nodes == nil {
				t.Error("GetNodes returned nil")
			}
		}()
	}

	// Test concurrent GetClient calls
	for i := 0; i < 10; i++ {
		go func(id int) {
			client, _ := discovery.GetClient(fmt.Sprintf("node-%d", id))
			_ = client // may be nil, that's fine
		}(i)
	}

	time.Sleep(100 * time.Millisecond) // Let concurrent operations complete
}
