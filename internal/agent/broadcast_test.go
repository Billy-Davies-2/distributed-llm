package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"distributed-llm/pkg/models"
)

// MockBroadcaster for testing
type MockBroadcaster struct {
	mu        sync.RWMutex
	resources models.ResourceInfo
	nodes     []models.Node
	listeners []chan models.ResourceInfo
}

func NewMockBroadcaster() *MockBroadcaster {
	return &MockBroadcaster{
		resources: models.ResourceInfo{
			CPUCores:   4,
			MemoryMB:   8192,
			MaxLayers:  8,
			UsedLayers: 0,
		},
		nodes:     []models.Node{},
		listeners: []chan models.ResourceInfo{},
	}
}

func (m *MockBroadcaster) Start(ctx context.Context) error {
	// Simulate periodic broadcasting
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.broadcast()
			}
		}
	}()
	return nil
}

func (m *MockBroadcaster) broadcast() {
	m.mu.RLock()
	resources := m.resources
	listeners := make([]chan models.ResourceInfo, len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.RUnlock()

	for _, listener := range listeners {
		select {
		case listener <- resources:
		default:
			// Non-blocking send
		}
	}
}

func (m *MockBroadcaster) Subscribe() <-chan models.ResourceInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan models.ResourceInfo, 10)
	m.listeners = append(m.listeners, ch)
	return ch
}

func (m *MockBroadcaster) UpdateResources(resources models.ResourceInfo) {
	m.mu.Lock()
	m.resources = resources
	m.mu.Unlock()
}

func (m *MockBroadcaster) AddNode(node models.Node) {
	m.mu.Lock()
	m.nodes = append(m.nodes, node)
	m.mu.Unlock()
}

func (m *MockBroadcaster) GetNodes() []models.Node {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nodes := make([]models.Node, len(m.nodes))
	copy(nodes, m.nodes)
	return nodes
}

func TestBroadcasterStart(t *testing.T) {
	broadcaster := NewMockBroadcaster()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := broadcaster.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start broadcaster: %v", err)
	}

	// Wait a bit to ensure it's running
	time.Sleep(50 * time.Millisecond)
}

func TestBroadcasterSubscription(t *testing.T) {
	broadcaster := NewMockBroadcaster()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start broadcaster
	err := broadcaster.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start broadcaster: %v", err)
	}

	// Subscribe to updates
	updates := broadcaster.Subscribe()

	// Update resources
	newResources := models.ResourceInfo{
		CPUCores:   8,
		MemoryMB:   16384,
		MaxLayers:  16,
		UsedLayers: 4,
	}
	broadcaster.UpdateResources(newResources)

	// Wait for update
	select {
	case received := <-updates:
		if received.CPUCores != newResources.CPUCores {
			t.Errorf("Expected CPUCores %d, got %d", newResources.CPUCores, received.CPUCores)
		}
		if received.MemoryMB != newResources.MemoryMB {
			t.Errorf("Expected MemoryMB %d, got %d", newResources.MemoryMB, received.MemoryMB)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Timeout waiting for resource update")
	}
}

func TestBroadcasterMultipleSubscribers(t *testing.T) {
	broadcaster := NewMockBroadcaster()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := broadcaster.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start broadcaster: %v", err)
	}

	// Create multiple subscribers
	const numSubscribers = 3
	subscribers := make([]<-chan models.ResourceInfo, numSubscribers)
	for i := 0; i < numSubscribers; i++ {
		subscribers[i] = broadcaster.Subscribe()
	}

	// Update resources
	newResources := models.ResourceInfo{
		CPUCores:   12,
		MemoryMB:   32768,
		MaxLayers:  32,
		UsedLayers: 10,
	}
	broadcaster.UpdateResources(newResources)

	// Verify all subscribers receive the update
	for i, sub := range subscribers {
		select {
		case received := <-sub:
			if received.CPUCores != newResources.CPUCores {
				t.Errorf("Subscriber %d: Expected CPUCores %d, got %d",
					i, newResources.CPUCores, received.CPUCores)
			}
		case <-time.After(500 * time.Millisecond):
			t.Errorf("Subscriber %d: Timeout waiting for update", i)
		}
	}
}

func TestBroadcasterNodeManagement(t *testing.T) {
	broadcaster := NewMockBroadcaster()

	// Add nodes
	nodes := []models.Node{
		{
			ID:      "node1",
			Address: "192.168.1.100",
			Port:    8080,
			Status:  models.NodeStatusOnline,
			Resources: models.ResourceInfo{
				CPUCores: 4,
				MemoryMB: 8192,
			},
		},
		{
			ID:      "node2",
			Address: "192.168.1.101",
			Port:    8080,
			Status:  models.NodeStatusOnline,
			Resources: models.ResourceInfo{
				CPUCores: 8,
				MemoryMB: 16384,
			},
		},
	}

	for _, node := range nodes {
		broadcaster.AddNode(node)
	}

	// Verify nodes were added
	retrievedNodes := broadcaster.GetNodes()
	if len(retrievedNodes) != len(nodes) {
		t.Errorf("Expected %d nodes, got %d", len(nodes), len(retrievedNodes))
	}

	// Verify node data
	for i, node := range retrievedNodes {
		if node.ID != nodes[i].ID {
			t.Errorf("Node %d: Expected ID %s, got %s", i, nodes[i].ID, node.ID)
		}
		if node.Address != nodes[i].Address {
			t.Errorf("Node %d: Expected Address %s, got %s", i, nodes[i].Address, node.Address)
		}
	}
}

func TestBroadcasterConcurrency(t *testing.T) {
	broadcaster := NewMockBroadcaster()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := broadcaster.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start broadcaster: %v", err)
	}

	const numGoroutines = 10
	const numUpdates = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines updating resources concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numUpdates; j++ {
				resources := models.ResourceInfo{
					CPUCores:   int64(id*10 + j),
					MemoryMB:   int64((id*1000 + j) * 1024),
					MaxLayers:  int32(id*10 + j),
					UsedLayers: int32(j % 10),
				}
				broadcaster.UpdateResources(resources)
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race conditions occur
}

// Benchmark tests
func BenchmarkBroadcast(b *testing.B) {
	broadcaster := NewMockBroadcaster()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broadcaster.Start(ctx)

	resources := models.ResourceInfo{
		CPUCores:   8,
		MemoryMB:   16384,
		MaxLayers:  16,
		UsedLayers: 4,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broadcaster.UpdateResources(resources)
	}
}

func BenchmarkSubscription(b *testing.B) {
	broadcaster := NewMockBroadcaster()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broadcaster.Start(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = broadcaster.Subscribe()
	}
}
