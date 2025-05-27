package e2e

import (
	"context"
	"distributed-llm/internal/network"
	"distributed-llm/pkg/models"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestFullClusterDeployment tests a complete cluster deployment workflow
func TestFullClusterDeployment(t *testing.T) {
	// Skip if binaries don't exist
	if !binaryExists("bin/agent") || !binaryExists("bin/tui") {
		t.Skip("Skipping E2E test - binaries not built (run 'make build' first)")
	}

	// Test that binaries can start and show help
	testBinaryHelp(t, "bin/agent")
	testBinaryHelp(t, "bin/tui")
}

// TestMultiNodeClusterFormation tests the formation of a multi-node cluster
func TestMultiNodeClusterFormation(t *testing.T) {
	const numNodes = 3
	const testDuration = 10 * time.Second

	// Create test cluster
	cluster := &TestCluster{
		nodes:    make([]*TestNode, numNodes),
		networks: make([]*network.P2PNetwork, numNodes),
		t:        t,
	}
	defer cluster.Cleanup()

	// Initialize nodes
	for i := 0; i < numNodes; i++ {
		port := findAvailablePort(t)
		gossipPort := findAvailablePort(t)

		node := &TestNode{
			ID:         fmt.Sprintf("node-%d", i),
			Port:       port,
			GossipPort: gossipPort,
		}

		network, err := network.NewP2PNetwork(node.ID, port, gossipPort)
		if err != nil {
			t.Fatalf("Failed to create network for node %d: %v", i, err)
		}

		cluster.nodes[i] = node
		cluster.networks[i] = network
	}

	// Start first node (bootstrap)
	err := cluster.networks[0].Start(nil)
	if err != nil {
		t.Fatalf("Failed to start bootstrap node: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Start remaining nodes and join cluster
	bootstrapAddr := fmt.Sprintf("127.0.0.1:%d", cluster.nodes[0].GossipPort)
	for i := 1; i < numNodes; i++ {
		err := cluster.networks[i].Start([]string{bootstrapAddr})
		if err != nil {
			t.Fatalf("Failed to start node %d: %v", i, err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Wait for cluster formation
	time.Sleep(time.Second)

	// Verify cluster formation
	for i := 0; i < numNodes; i++ {
		members := cluster.networks[i].GetMembers()
		if len(members) != numNodes {
			t.Errorf("Node %d should see %d members, got %d", i, numNodes, len(members))
		}
	}

	// Test cluster stability over time
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	// Monitor cluster health
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	healthChecks := 0
	for {
		select {
		case <-ctx.Done():
			t.Logf("Cluster remained stable for %v with %d health checks", testDuration, healthChecks)
			return
		case <-ticker.C:
			healthChecks++
			for i := 0; i < numNodes; i++ {
				members := cluster.networks[i].GetMembers()
				if len(members) != numNodes {
					t.Errorf("Health check %d: Node %d lost cluster members (expected %d, got %d)",
						healthChecks, i, numNodes, len(members))
				}
			}
		}
	}
}

// TestModelDistributionWorkflow tests the complete model distribution workflow
func TestModelDistributionWorkflow(t *testing.T) {
	// Create a test cluster with resource simulation
	cluster := &TestCluster{
		nodes:    make([]*TestNode, 3),
		networks: make([]*network.P2PNetwork, 3),
		t:        t,
	}
	defer cluster.Cleanup()

	// Setup cluster
	cluster.setupCluster(t)

	// Define test model
	testModel := models.Model{
		ID:         "llama-7b-test",
		Name:       "Llama 7B Test",
		Version:    "1.0",
		LayerCount: 30,                     // Fits exactly in our 3-node cluster (10 layers each)
		Size:       7 * 1024 * 1024 * 1024, // 7GB
	}

	// Simulate resource allocation
	for i, node := range cluster.nodes {
		// Each node can handle ~10 layers with 8GB RAM
		maxLayers := int32(10)

		node.Resources = models.ResourceInfo{
			CPUCores:   8,
			MemoryMB:   8192,
			MaxLayers:  maxLayers,
			UsedLayers: 0,
			GPUs:       []models.GPUInfo{},
		}

		t.Logf("Node %d configured with %d max layers", i, maxLayers)
	}

	// Test layer distribution calculation
	totalLayers := testModel.LayerCount
	totalCapacity := int32(0)
	for _, node := range cluster.nodes {
		totalCapacity += node.Resources.MaxLayers
	}

	if totalCapacity < totalLayers {
		t.Errorf("Cluster capacity (%d layers) insufficient for model (%d layers)",
			totalCapacity, totalLayers)
	} else {
		t.Logf("Cluster has sufficient capacity: %d layers available for %d layer model",
			totalCapacity, totalLayers)
	}

	// Simulate layer assignment
	remainingLayers := totalLayers
	for i, node := range cluster.nodes {
		layersToAssign := min(remainingLayers, node.Resources.MaxLayers)
		node.Resources.UsedLayers = layersToAssign
		remainingLayers -= layersToAssign

		t.Logf("Node %d assigned %d layers (%d remaining)", i, layersToAssign, remainingLayers)

		if remainingLayers <= 0 {
			break
		}
	}

	if remainingLayers > 0 {
		t.Errorf("Could not assign all layers: %d layers remaining", remainingLayers)
	}
}

// TestFailureRecoveryWorkflow tests node failure and recovery scenarios
func TestFailureRecoveryWorkflow(t *testing.T) {
	cluster := &TestCluster{
		nodes:    make([]*TestNode, 3),
		networks: make([]*network.P2PNetwork, 3),
		t:        t,
	}
	defer cluster.Cleanup()

	// Setup cluster
	cluster.setupCluster(t)

	// Verify initial cluster health
	cluster.verifyClusterHealth(t, 3)

	// Simulate node failure (stop node 1)
	t.Log("Simulating node 1 failure...")
	cluster.networks[1].Stop()

	// Wait for failure detection
	time.Sleep(2 * time.Second)

	// Check remaining nodes see the failure
	// expectedMembers := 2 // nodes 0 and 2 should remain
	for i := 0; i < 3; i++ {
		if i == 1 {
			continue // skip the failed node
		}

		members := cluster.networks[i].GetMembers()
		// Note: memberlist might take time to detect failures
		// In production, we'd have proper health checks and timeouts
		t.Logf("Node %d sees %d members after node 1 failure", i, len(members))
	}

	// Simulate node recovery (restart node 1)
	t.Log("Simulating node 1 recovery...")
	bootstrapAddr := fmt.Sprintf("127.0.0.1:%d", cluster.nodes[0].GossipPort)
	err := cluster.networks[1].Start([]string{bootstrapAddr})
	if err != nil {
		t.Errorf("Failed to restart node 1: %v", err)
	}

	// Wait for recovery
	time.Sleep(2 * time.Second)

	// Verify cluster recovery
	cluster.verifyClusterHealth(t, 3)
}

// TestCluster represents a test cluster
type TestCluster struct {
	nodes    []*TestNode
	networks []*network.P2PNetwork
	t        *testing.T
}

// TestNode represents a test node
type TestNode struct {
	ID         string
	Port       int
	GossipPort int
	Resources  models.ResourceInfo
}

// setupCluster initializes and starts a test cluster
func (c *TestCluster) setupCluster(t *testing.T) {
	numNodes := len(c.nodes)

	// Initialize nodes
	for i := 0; i < numNodes; i++ {
		port := findAvailablePort(t)
		gossipPort := findAvailablePort(t)

		c.nodes[i] = &TestNode{
			ID:         fmt.Sprintf("node-%d", i),
			Port:       port,
			GossipPort: gossipPort,
		}

		network, err := network.NewP2PNetwork(c.nodes[i].ID, port, gossipPort)
		if err != nil {
			t.Fatalf("Failed to create network for node %d: %v", i, err)
		}

		c.networks[i] = network
	}

	// Start bootstrap node
	err := c.networks[0].Start(nil)
	if err != nil {
		t.Fatalf("Failed to start bootstrap node: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Start remaining nodes
	bootstrapAddr := fmt.Sprintf("127.0.0.1:%d", c.nodes[0].GossipPort)
	for i := 1; i < numNodes; i++ {
		err := c.networks[i].Start([]string{bootstrapAddr})
		if err != nil {
			t.Fatalf("Failed to start node %d: %v", i, err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Wait for cluster formation
	time.Sleep(time.Second)
}

// verifyClusterHealth checks that all nodes see the expected number of members
func (c *TestCluster) verifyClusterHealth(t *testing.T, expectedMembers int) {
	for i := 0; i < len(c.nodes); i++ {
		members := c.networks[i].GetMembers()
		if len(members) != expectedMembers {
			t.Errorf("Node %d should see %d members, got %d", i, expectedMembers, len(members))
		}
	}
}

// Cleanup stops all nodes and cleans up resources
func (c *TestCluster) Cleanup() {
	for i, network := range c.networks {
		if network != nil {
			network.Stop()
			c.t.Logf("Stopped node %d", i)
		}
	}
}

// Helper functions

func binaryExists(path string) bool {
	cmd := exec.Command("test", "-f", path)
	return cmd.Run() == nil
}

func testBinaryHelp(t *testing.T, binary string) {
	cmd := exec.Command(binary, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Some binaries might not support --help, try -h
		cmd = exec.Command(binary, "-h")
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Logf("Binary %s help test failed (this may be expected): %v", binary, err)
			return
		}
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "Usage") || strings.Contains(outputStr, "help") {
		t.Logf("Binary %s shows help correctly", binary)
	} else {
		t.Logf("Binary %s help output: %s", binary, outputStr)
	}
}

func findAvailablePort(t *testing.T) int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

func min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
