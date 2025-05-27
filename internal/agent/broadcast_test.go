package agent

import (
	"context"
	"fmt"
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

// MockMetricsCollector for testing the real Broadcaster with metrics
type MockMetricsCollector struct {
	mu                   sync.Mutex
	nodeResourcesUpdates []models.ResourceInfo
	nodeStatusUpdates    []models.NodeStatus
	networkConnections   []int
	networkMessages      []NetworkMessage
}

type NetworkMessage struct {
	Direction   string
	MessageType string
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		nodeResourcesUpdates: []models.ResourceInfo{},
		nodeStatusUpdates:    []models.NodeStatus{},
		networkConnections:   []int{},
		networkMessages:      []NetworkMessage{},
	}
}

func (m *MockMetricsCollector) UpdateNodeResources(resources models.ResourceInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeResourcesUpdates = append(m.nodeResourcesUpdates, resources)
}

func (m *MockMetricsCollector) UpdateNodeStatus(status models.NodeStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeStatusUpdates = append(m.nodeStatusUpdates, status)
}

func (m *MockMetricsCollector) UpdateNetworkConnections(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.networkConnections = append(m.networkConnections, count)
}

func (m *MockMetricsCollector) RecordNetworkMessage(direction, messageType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.networkMessages = append(m.networkMessages, NetworkMessage{
		Direction:   direction,
		MessageType: messageType,
	})
}

func (m *MockMetricsCollector) GetNodeResourcesUpdates() []models.ResourceInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]models.ResourceInfo, len(m.nodeResourcesUpdates))
	copy(result, m.nodeResourcesUpdates)
	return result
}

func (m *MockMetricsCollector) GetNetworkMessages() []NetworkMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]NetworkMessage, len(m.networkMessages))
	copy(result, m.networkMessages)
	return result
}

func (m *MockMetricsCollector) GetNetworkConnections() []int {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]int, len(m.networkConnections))
	copy(result, m.networkConnections)
	return result
}

// Tests for the real Broadcaster implementation
func TestRealBroadcasterNewBroadcaster(t *testing.T) {
	broadcaster := NewBroadcaster()

	if broadcaster == nil {
		t.Fatal("NewBroadcaster returned nil")
	}

	if len(broadcaster.listeners) != 0 {
		t.Errorf("Expected empty listeners, got %d", len(broadcaster.listeners))
	}

	if len(broadcaster.nodes) != 0 {
		t.Errorf("Expected empty nodes, got %d", len(broadcaster.nodes))
	}

	// Verify initial resources are populated
	resources := broadcaster.GetResources()
	if resources.CPUCores <= 0 {
		t.Error("Expected positive CPU cores from GetResourceInfo()")
	}
}

func TestRealBroadcasterSetMetricsCollector(t *testing.T) {
	broadcaster := NewBroadcaster()
	mockCollector := NewMockMetricsCollector()

	broadcaster.SetMetricsCollector(mockCollector)

	// Verify metrics collector is set by triggering an operation that uses it
	resources := models.ResourceInfo{
		CPUCores:   8,
		MemoryMB:   16384,
		MaxLayers:  16,
		UsedLayers: 4,
	}

	broadcaster.UpdateResources(resources)

	// Check that metrics were recorded
	updates := mockCollector.GetNodeResourcesUpdates()
	if len(updates) != 1 {
		t.Fatalf("Expected 1 resource update, got %d", len(updates))
	}

	if updates[0].CPUCores != resources.CPUCores {
		t.Errorf("Expected CPUCores %d, got %d", resources.CPUCores, updates[0].CPUCores)
	}
}

func TestRealBroadcasterStart(t *testing.T) {
	broadcaster := NewBroadcaster()
	mockCollector := NewMockMetricsCollector()
	broadcaster.SetMetricsCollector(mockCollector)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := broadcaster.Start(ctx)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	// Wait for context to complete
	<-ctx.Done()

	// The broadcaster should have started successfully (no error means success)
	// We can't easily test the ticker behavior without waiting 30 seconds
}

func TestRealBroadcasterStartWithBroadcast(t *testing.T) {
	broadcaster := NewBroadcaster()
	mockCollector := NewMockMetricsCollector()
	broadcaster.SetMetricsCollector(mockCollector)

	// Create a subscription channel
	updateCh := make(chan models.ResourceInfo, 10)
	broadcaster.Subscribe(updateCh)

	// Manually trigger broadcast to test it works
	broadcaster.broadcast()

	// Check that broadcast was recorded in metrics
	messages := mockCollector.GetNetworkMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 network message, got %d", len(messages))
	}

	if messages[0].Direction != "outgoing" || messages[0].MessageType != "resource_broadcast" {
		t.Errorf("Expected outgoing resource_broadcast, got %s %s",
			messages[0].Direction, messages[0].MessageType)
	}

	// Verify subscriber received the update
	select {
	case received := <-updateCh:
		// Should receive the initial resources from GetResourceInfo()
		if received.CPUCores <= 0 {
			t.Error("Expected positive CPU cores in broadcast")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for broadcast")
	}
}

func TestRealBroadcasterUpdateResources(t *testing.T) {
	broadcaster := NewBroadcaster()
	mockCollector := NewMockMetricsCollector()
	broadcaster.SetMetricsCollector(mockCollector)

	// Create subscribers
	ch1 := make(chan models.ResourceInfo, 10)
	ch2 := make(chan models.ResourceInfo, 10)
	broadcaster.Subscribe(ch1)
	broadcaster.Subscribe(ch2)

	newResources := models.ResourceInfo{
		CPUCores:   12,
		MemoryMB:   32768,
		MaxLayers:  32,
		UsedLayers: 8,
		GPUs: []models.GPUInfo{
			{Name: "NVIDIA RTX 4090", MemoryMB: 8192, UUID: "gpu-uuid-1"},
		},
	}

	broadcaster.UpdateResources(newResources)

	// Verify metrics were updated
	updates := mockCollector.GetNodeResourcesUpdates()
	if len(updates) != 1 {
		t.Fatalf("Expected 1 resource update, got %d", len(updates))
	}

	if updates[0].CPUCores != newResources.CPUCores {
		t.Errorf("Expected CPUCores %d, got %d", newResources.CPUCores, updates[0].CPUCores)
	}

	// Verify both subscribers received the update
	for i, ch := range []chan models.ResourceInfo{ch1, ch2} {
		select {
		case received := <-ch:
			if received.CPUCores != newResources.CPUCores {
				t.Errorf("Subscriber %d: Expected CPUCores %d, got %d",
					i, newResources.CPUCores, received.CPUCores)
			}
			if len(received.GPUs) != 1 {
				t.Errorf("Subscriber %d: Expected 1 GPU, got %d", i, len(received.GPUs))
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Subscriber %d: Timeout waiting for update", i)
		}
	}
}

func TestRealBroadcasterGetResources(t *testing.T) {
	broadcaster := NewBroadcaster()

	// Get initial resources
	initialResources := broadcaster.GetResources()
	if initialResources.CPUCores <= 0 {
		t.Error("Expected positive CPU cores from initial resources")
	}

	// Update resources
	newResources := models.ResourceInfo{
		CPUCores:   16,
		MemoryMB:   65536,
		MaxLayers:  64,
		UsedLayers: 16,
	}

	broadcaster.UpdateResources(newResources)

	// Get updated resources
	updatedResources := broadcaster.GetResources()
	if updatedResources.CPUCores != newResources.CPUCores {
		t.Errorf("Expected CPUCores %d, got %d", newResources.CPUCores, updatedResources.CPUCores)
	}

	if updatedResources.MemoryMB != newResources.MemoryMB {
		t.Errorf("Expected MemoryMB %d, got %d", newResources.MemoryMB, updatedResources.MemoryMB)
	}
}

func TestRealBroadcasterSubscribe(t *testing.T) {
	broadcaster := NewBroadcaster()

	// Initial state should have no listeners
	if len(broadcaster.listeners) != 0 {
		t.Errorf("Expected 0 initial listeners, got %d", len(broadcaster.listeners))
	}

	// Add first subscriber
	ch1 := make(chan models.ResourceInfo, 10)
	broadcaster.Subscribe(ch1)

	if len(broadcaster.listeners) != 1 {
		t.Errorf("Expected 1 listener after first subscribe, got %d", len(broadcaster.listeners))
	}

	// Add second subscriber
	ch2 := make(chan models.ResourceInfo, 10)
	broadcaster.Subscribe(ch2)

	if len(broadcaster.listeners) != 2 {
		t.Errorf("Expected 2 listeners after second subscribe, got %d", len(broadcaster.listeners))
	}
}

func TestRealBroadcasterAddNode(t *testing.T) {
	broadcaster := NewBroadcaster()
	mockCollector := NewMockMetricsCollector()
	broadcaster.SetMetricsCollector(mockCollector)

	node1 := models.Node{
		ID:      "test-node-1",
		Address: "192.168.1.100",
		Port:    8080,
		Status:  models.NodeStatusOnline,
		Resources: models.ResourceInfo{
			CPUCores: 8,
			MemoryMB: 16384,
		},
	}

	node2 := models.Node{
		ID:      "test-node-2",
		Address: "192.168.1.101",
		Port:    8080,
		Status:  models.NodeStatusOnline,
		Resources: models.ResourceInfo{
			CPUCores: 16,
			MemoryMB: 32768,
		},
	}

	// Add first node
	broadcaster.AddNode(node1)

	// Verify metrics were updated
	connections := mockCollector.GetNetworkConnections()
	if len(connections) != 1 || connections[0] != 1 {
		t.Errorf("Expected 1 connection update with value 1, got %v", connections)
	}

	messages := mockCollector.GetNetworkMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 network message, got %d", len(messages))
	}

	if messages[0].Direction != "incoming" || messages[0].MessageType != "node_added" {
		t.Errorf("Expected incoming node_added, got %s %s",
			messages[0].Direction, messages[0].MessageType)
	}

	// Add second node
	broadcaster.AddNode(node2)

	// Verify updated metrics
	connections = mockCollector.GetNetworkConnections()
	if len(connections) != 2 || connections[1] != 2 {
		t.Errorf("Expected 2 connection updates, last with value 2, got %v", connections)
	}
}

func TestRealBroadcasterGetNodes(t *testing.T) {
	broadcaster := NewBroadcaster()

	// Initially should have no nodes
	nodes := broadcaster.GetNodes()
	if len(nodes) != 0 {
		t.Errorf("Expected 0 initial nodes, got %d", len(nodes))
	}

	// Add some nodes
	node1 := models.Node{ID: "node1", Address: "addr1", Port: 8080, Status: models.NodeStatusOnline}
	node2 := models.Node{ID: "node2", Address: "addr2", Port: 8080, Status: models.NodeStatusOnline}

	broadcaster.AddNode(node1)
	broadcaster.AddNode(node2)

	// Get nodes and verify
	nodes = broadcaster.GetNodes()
	if len(nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(nodes))
	}

	// Verify node data
	if nodes[0].ID != node1.ID {
		t.Errorf("Expected first node ID %s, got %s", node1.ID, nodes[0].ID)
	}

	if nodes[1].ID != node2.ID {
		t.Errorf("Expected second node ID %s, got %s", node2.ID, nodes[1].ID)
	}

	// Verify returned slice is a copy (modifications shouldn't affect original)
	nodes[0].ID = "modified"
	originalNodes := broadcaster.GetNodes()
	if originalNodes[0].ID == "modified" {
		t.Error("GetNodes should return a copy, not the original slice")
	}
}

func TestRealBroadcasterConcurrentAccess(t *testing.T) {
	broadcaster := NewBroadcaster()
	mockCollector := NewMockMetricsCollector()
	broadcaster.SetMetricsCollector(mockCollector)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := broadcaster.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start broadcaster: %v", err)
	}

	const numGoroutines = 10
	const numOperations = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 types of operations

	// Concurrent resource updates
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
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

	// Concurrent node additions
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				node := models.Node{
					ID:      fmt.Sprintf("node-%d-%d", id, j),
					Address: fmt.Sprintf("192.168.%d.%d", id, j),
					Port:    8080,
					Status:  models.NodeStatusOnline,
				}
				broadcaster.AddNode(node)
			}
		}(i)
	}

	// Concurrent subscriptions
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				ch := make(chan models.ResourceInfo, 100)
				broadcaster.Subscribe(ch)
			}
		}()
	}

	wg.Wait()

	// Verify final state
	nodes := broadcaster.GetNodes()
	expectedNodes := numGoroutines * numOperations
	if len(nodes) != expectedNodes {
		t.Errorf("Expected %d nodes after concurrent operations, got %d", expectedNodes, len(nodes))
	}

	// Verify metrics were collected
	resourceUpdates := mockCollector.GetNodeResourcesUpdates()
	if len(resourceUpdates) != numGoroutines*numOperations {
		t.Errorf("Expected %d resource updates, got %d",
			numGoroutines*numOperations, len(resourceUpdates))
	}
}

func TestRealBroadcasterSubscribeWithFullChannel(t *testing.T) {
	broadcaster := NewBroadcaster()

	// Create a channel with capacity 1 and fill it
	ch := make(chan models.ResourceInfo, 1)
	ch <- models.ResourceInfo{CPUCores: 1} // Fill the channel

	broadcaster.Subscribe(ch)

	// Update resources - should not block even though channel is full
	newResources := models.ResourceInfo{CPUCores: 8, MemoryMB: 16384}

	done := make(chan bool)
	go func() {
		broadcaster.UpdateResources(newResources)
		done <- true
	}()

	// Should complete quickly without blocking
	select {
	case <-done:
		// Success - UpdateResources completed without blocking
	case <-time.After(100 * time.Millisecond):
		t.Error("UpdateResources blocked on full channel")
	}
}

// Benchmark tests for the real Broadcaster
func BenchmarkRealBroadcasterUpdateResources(b *testing.B) {
	broadcaster := NewBroadcaster()
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

func BenchmarkRealBroadcasterSubscribe(b *testing.B) {
	broadcaster := NewBroadcaster()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := make(chan models.ResourceInfo, 10)
		broadcaster.Subscribe(ch)
	}
}

func BenchmarkRealBroadcasterAddNode(b *testing.B) {
	broadcaster := NewBroadcaster()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node := models.Node{
			ID:      fmt.Sprintf("node-%d", i),
			Address: fmt.Sprintf("192.168.1.%d", i%254+1),
			Port:    8080,
			Status:  models.NodeStatusOnline,
		}
		broadcaster.AddNode(node)
	}
}
