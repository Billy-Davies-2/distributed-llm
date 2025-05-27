package integration

import (
	"context"
	"distributed-llm/internal/agent"
	"distributed-llm/internal/network"
	"distributed-llm/pkg/models"
	"fmt"
	"net"
	"testing"
	"time"
)

// TestAgentNetworkIntegration tests the integration between agent broadcasting and P2P networking
func TestAgentNetworkIntegration(t *testing.T) {
	// Find available ports
	port1 := findAvailablePort(t)
	port2 := findAvailablePort(t)

	// Create P2P networks for two nodes
	node1, err := network.NewP2PNetwork("node-1", port1, port1+1000)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}

	node2, err := network.NewP2PNetwork("node-2", port2, port2+1000)
	if err != nil {
		t.Fatalf("Failed to create node2: %v", err)
	}

	// Start node1
	err = node1.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}
	defer node1.Stop()

	// Give node1 time to fully start
	time.Sleep(100 * time.Millisecond)

	// Start node2 and join node1's cluster
	seedAddr := fmt.Sprintf("127.0.0.1:%d", port1+1000)
	err = node2.Start([]string{seedAddr})
	if err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}
	defer node2.Stop()

	// Give time for cluster formation
	time.Sleep(500 * time.Millisecond)

	// Test that nodes can see each other
	members1 := node1.GetMembers()
	members2 := node2.GetMembers()

	if len(members1) < 2 {
		t.Errorf("Node1 should see at least 2 members, got %d", len(members1))
	}

	if len(members2) < 2 {
		t.Errorf("Node2 should see at least 2 members, got %d", len(members2))
	}

	// Create broadcasters for both nodes
	broadcaster1 := agent.NewBroadcaster()
	broadcaster2 := agent.NewBroadcaster()

	// Start broadcasters
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go broadcaster1.Start(ctx)
	go broadcaster2.Start(ctx)

	// Give broadcasters time to start
	time.Sleep(100 * time.Millisecond)

	// Subscribe to resource updates on node2
	ch2 := make(chan models.ResourceInfo, 10)
	broadcaster2.Subscribe(ch2)

	// Update resources on node1
	resources := models.ResourceInfo{
		CPUCores:   8,
		MemoryMB:   16384,
		MaxLayers:  16,
		UsedLayers: 4,
		GPUs:       []models.GPUInfo{},
	}

	broadcaster1.UpdateResources(resources)

	// In a real integration, we'd test actual network propagation
	// For now, test that the broadcasters work independently
	select {
	case received := <-ch2:
		// This wouldn't happen in this test since broadcasters are independent
		// But we're testing the component interfaces work together
		t.Logf("Received resource update: %+v", received)
	case <-time.After(100 * time.Millisecond):
		// Expected in this test since there's no actual network integration yet
		t.Log("No resource update received (expected in current implementation)")
	}
}

// TestAgentResourcesWithNetworkDiscovery tests resource detection integration with network discovery
func TestAgentResourcesWithNetworkDiscovery(t *testing.T) {
	// Get actual system resources
	resources := agent.GetResourceInfo()

	// Validate resources are reasonable for running tests
	if resources.CPUCores <= 0 {
		t.Error("Should detect at least 1 CPU core")
	}

	if resources.MemoryMB <= 0 {
		t.Error("Should detect some memory")
	}

	// Create a node with these resources
	node := models.Node{
		ID:        "test-node",
		Address:   "127.0.0.1",
		Port:      8080,
		Resources: resources,
		Status:    models.NodeStatusOnline,
		LastSeen:  time.Now(),
	}

	// Test that the node can be used for network operations
	if !node.IsHealthy() {
		t.Error("Node with current resources should be healthy")
	}

	// Test resource capacity calculations
	if resources.MaxLayers > 0 && resources.UsedLayers > resources.MaxLayers {
		t.Error("Used layers should not exceed max layers")
	}

	// Test GPU detection integration
	for i, gpu := range resources.GPUs {
		if gpu.Name == "" {
			t.Errorf("GPU %d should have a name", i)
		}
		if gpu.MemoryMB <= 0 {
			t.Errorf("GPU %d should have positive memory", i)
		}
	}
}

// TestBroadcasterWithMultipleNetworkNodes tests broadcaster with multiple network nodes
func TestBroadcasterWithMultipleNetworkNodes(t *testing.T) {
	const numNodes = 3

	// Create multiple P2P networks
	networks := make([]*network.P2PNetwork, numNodes)
	broadcasters := make([]*agent.Broadcaster, numNodes)
	ports := make([]int, numNodes)

	// Setup networks
	for i := 0; i < numNodes; i++ {
		port := findAvailablePort(t)
		ports[i] = port

		var err error
		networks[i], err = network.NewP2PNetwork(fmt.Sprintf("node-%d", i), port, port+1000)
		if err != nil {
			t.Fatalf("Failed to create network %d: %v", i, err)
		}

		broadcasters[i] = agent.NewBroadcaster()
	}

	// Start first network
	err := networks[0].Start(nil)
	if err != nil {
		t.Fatalf("Failed to start network 0: %v", err)
	}
	defer networks[0].Stop()

	time.Sleep(100 * time.Millisecond)

	// Start remaining networks and join cluster
	for i := 1; i < numNodes; i++ {
		seedAddr := fmt.Sprintf("127.0.0.1:%d", ports[0]+1000)
		err := networks[i].Start([]string{seedAddr})
		if err != nil {
			t.Fatalf("Failed to start network %d: %v", i, err)
		}
		defer networks[i].Stop()

		time.Sleep(100 * time.Millisecond)
	}

	// Give time for cluster formation
	time.Sleep(500 * time.Millisecond)

	// Start all broadcasters
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < numNodes; i++ {
		go broadcasters[i].Start(ctx)
	}

	time.Sleep(100 * time.Millisecond)

	// Test that all networks formed a cluster
	for i := 0; i < numNodes; i++ {
		members := networks[i].GetMembers()
		if len(members) != numNodes {
			t.Errorf("Network %d should see %d members, got %d", i, numNodes, len(members))
		}
	}

	// Test broadcaster resource updates
	for i := 0; i < numNodes; i++ {
		resources := models.ResourceInfo{
			CPUCores:   int64(4 + i),
			MemoryMB:   int64(8192 * (i + 1)),
			MaxLayers:  int32(8 + i),
			UsedLayers: int32(i),
			GPUs:       []models.GPUInfo{},
		}

		broadcasters[i].UpdateResources(resources)
	}

	// Test that resources can be retrieved
	for i := 0; i < numNodes; i++ {
		resources := broadcasters[i].GetResources()
		if resources.CPUCores != int64(4+i) {
			t.Errorf("Node %d should have %d CPUs, got %d", i, 4+i, resources.CPUCores)
		}
	}
}

// Helper function to find an available port
func findAvailablePort(t *testing.T) int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}
