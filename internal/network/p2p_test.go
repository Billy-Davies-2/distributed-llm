package network

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestP2PNetworkCreation(t *testing.T) {
	network, err := NewP2PNetwork("test-node", 8080, 7946)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}

	if network == nil {
		t.Fatal("Network should not be nil")
	}
}

func TestP2PNetworkInvalidPorts(t *testing.T) {
	tests := []struct {
		name       string
		grpcPort   int
		gossipPort int
		shouldFail bool
	}{
		{"Valid ports", 8080, 7946, false},
		{"Invalid gRPC port", -1, 7946, true},
		{"Invalid gossip port", 8080, -1, true},
		{"Same ports", 8080, 8080, true},
		{"Reserved port", 80, 7946, false}, // Might fail due to permissions
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewP2PNetwork("test-node", tt.grpcPort, tt.gossipPort)
			if tt.shouldFail && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestP2PNetworkStart(t *testing.T) {
	// Find available ports
	grpcPort := findAvailablePort(t)
	gossipPort := findAvailablePort(t)

	network, err := NewP2PNetwork("test-node", grpcPort, gossipPort)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}

	// Test starting without seeds
	err = network.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start network: %v", err)
	}

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Stop the network
	network.Stop()
}

func TestP2PNetworkStartWithSeeds(t *testing.T) {
	grpcPort := findAvailablePort(t)
	gossipPort := findAvailablePort(t)

	network, err := NewP2PNetwork("test-node", grpcPort, gossipPort)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}

	// Test starting with seeds (even if they're unreachable)
	seeds := []string{"127.0.0.1:9999", "127.0.0.1:9998"}
	err = network.Start(seeds)
	if err != nil {
		t.Fatalf("Failed to start network with seeds: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	network.Stop()
}

func TestP2PNetworkMultipleNodes(t *testing.T) {
	const numNodes = 3
	networks := make([]*P2PNetwork, numNodes)
	ports := make([]int, numNodes*2) // gRPC and gossip ports for each node

	// Find available ports for all nodes
	for i := 0; i < numNodes*2; i++ {
		ports[i] = findAvailablePort(t)
	}

	// Create networks
	for i := 0; i < numNodes; i++ {
		grpcPort := ports[i*2]
		gossipPort := ports[i*2+1]

		network, err := NewP2PNetwork(
			fmt.Sprintf("node-%d", i),
			grpcPort,
			gossipPort,
		)
		if err != nil {
			t.Fatalf("Failed to create network %d: %v", i, err)
		}
		networks[i] = network
	}

	// Start first node without seeds
	err := networks[0].Start(nil)
	if err != nil {
		t.Fatalf("Failed to start first network: %v", err)
	}

	// Start other nodes with first node as seed
	seedAddr := fmt.Sprintf("127.0.0.1:%d", ports[1]) // First node's gossip port
	for i := 1; i < numNodes; i++ {
		err := networks[i].Start([]string{seedAddr})
		if err != nil {
			t.Fatalf("Failed to start network %d: %v", i, err)
		}
	}

	// Give time for nodes to discover each other
	time.Sleep(500 * time.Millisecond)

	// Stop all networks
	for i := numNodes - 1; i >= 0; i-- {
		networks[i].Stop()
	}
}

func TestP2PNetworkStop(t *testing.T) {
	grpcPort := findAvailablePort(t)
	gossipPort := findAvailablePort(t)

	network, err := NewP2PNetwork("test-node", grpcPort, gossipPort)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}

	// Start and then stop
	err = network.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start network: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Stop should not hang or panic
	network.Stop()

	// Multiple stops should be safe
	network.Stop()
	network.Stop()
}

func TestP2PNetworkConcurrentOperations(t *testing.T) {
	grpcPort := findAvailablePort(t)
	gossipPort := findAvailablePort(t)

	network, err := NewP2PNetwork("test-node", grpcPort, gossipPort)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}

	err = network.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start network: %v", err)
	}

	// Test concurrent operations
	done := make(chan bool, 2)

	// Goroutine 1: Try to stop the network
	go func() {
		time.Sleep(200 * time.Millisecond)
		network.Stop()
		done <- true
	}()

	// Goroutine 2: Try to perform operations
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(50 * time.Millisecond)
			// Could add network operations here
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

// Helper function to find an available port
func findAvailablePort(t *testing.T) int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port
}

// Benchmark tests
func BenchmarkP2PNetworkCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewP2PNetwork("test-node", 8080, 7946)
		if err != nil {
			b.Fatalf("Failed to create network: %v", err)
		}
	}
}

func BenchmarkP2PNetworkStartStop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		grpcPort := 8080 + i
		gossipPort := 7946 + i

		network, err := NewP2PNetwork("test-node", grpcPort, gossipPort)
		if err != nil {
			b.Fatalf("Failed to create network: %v", err)
		}

		err = network.Start(nil)
		if err != nil {
			b.Fatalf("Failed to start network: %v", err)
		}

		network.Stop()
	}
}
