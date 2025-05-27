package network

import (
	"context"
	"testing"
	"time"

	pb "distributed-llm/proto"
	"distributed-llm/pkg/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDiscoveryServer_DiscoverNodes(t *testing.T) {
	// Create a test P2P network
	network, err := NewP2PNetwork("test-node", 8080, 7946)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}
	defer network.Stop()

	// Add some test nodes
	testNodes := []models.Node{
		{
			ID:      "node-1",
			Address: "localhost",
			Port:    8080,
			Status:  models.NodeStatusOnline,
			Resources: models.ResourceInfo{
				CPUCores: 4,
				MemoryMB: 8192,
				GPUs: []models.GPUInfo{
					{Name: "RTX 4090", MemoryMB: 24576, UUID: "gpu-1"},
				},
				MaxLayers: 32,
			},
			LastSeen: time.Now(),
		},
		{
			ID:      "node-2",
			Address: "localhost",
			Port:    8081,
			Status:  models.NodeStatusBusy,
			Resources: models.ResourceInfo{
				CPUCores: 8,
				MemoryMB: 16384,
				MaxLayers: 64,
			},
			LastSeen: time.Now(),
		},
	}

	// Set nodes in network (we'll need to add a method for this)
	network.nodes = testNodes

	server := NewDiscoveryServer(network)

	req := &pb.DiscoveryRequest{
		RequesterId: "test-client",
		KnownNodes:  []string{"node-1"}, // Already know node-1
	}

	resp, err := server.DiscoverNodes(context.Background(), req)
	if err != nil {
		t.Fatalf("DiscoverNodes failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected successful discovery")
	}

	// Should only return node-2 since node-1 is already known
	if len(resp.DiscoveredNodes) != 1 {
		t.Errorf("Expected 1 discovered node, got %d", len(resp.DiscoveredNodes))
	}

	if resp.DiscoveredNodes[0].NodeId != "node-2" {
		t.Errorf("Expected node-2, got %s", resp.DiscoveredNodes[0].NodeId)
	}

	if resp.DiscoveredNodes[0].Status != "busy" {
		t.Errorf("Expected status 'busy', got %s", resp.DiscoveredNodes[0].Status)
	}
}

func TestDiscoveryServer_RegisterWithCluster(t *testing.T) {
	network, err := NewP2PNetwork("test-node", 8080, 7946)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}
	defer network.Stop()

	server := NewDiscoveryServer(network)

	req := &pb.ClusterJoinRequest{
		NodeId:    "new-node",
		Address:   "localhost",
		Port:      8082,
		SeedNodes: []string{"localhost:8080"},
	}

	resp, err := server.RegisterWithCluster(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterWithCluster failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected successful cluster join")
	}

	if resp.ClusterId != "distributed-llm-cluster" {
		t.Errorf("Expected cluster ID 'distributed-llm-cluster', got %s", resp.ClusterId)
	}

	if resp.Message == "" {
		t.Error("Expected non-empty message")
	}
}

func TestDiscoveryServer_LeaveCluster(t *testing.T) {
	network, err := NewP2PNetwork("test-node", 8080, 7946)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}
	defer network.Stop()

	server := NewDiscoveryServer(network)

	req := &pb.ClusterLeaveRequest{
		NodeId: "leaving-node",
		Reason: "Maintenance shutdown",
	}

	resp, err := server.LeaveCluster(context.Background(), req)
	if err != nil {
		t.Fatalf("LeaveCluster failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected successful cluster leave")
	}

	if resp.Message == "" {
		t.Error("Expected non-empty message")
	}
}

func TestDiscoveryServer_GetClusterInfo(t *testing.T) {
	network, err := NewP2PNetwork("test-node", 8080, 7946)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}
	defer network.Stop()

	// Add test nodes with different statuses
	testNodes := []models.Node{
		{
			ID:      "node-1",
			Status:  models.NodeStatusOnline,
			Resources: models.ResourceInfo{
				CPUCores:   4,
				MemoryMB:   8192,
				MaxLayers:  32,
				UsedLayers: 16,
			},
			LastSeen: time.Now(),
		},
		{
			ID:      "node-2",
			Status:  models.NodeStatusOffline,
			Resources: models.ResourceInfo{
				CPUCores:   8,
				MemoryMB:   16384,
				MaxLayers:  64,
				UsedLayers: 0,
			},
			LastSeen: time.Now().Add(-10 * time.Minute),
		},
	}

	network.nodes = testNodes

	server := NewDiscoveryServer(network)

	req := &pb.ClusterInfoRequest{
		RequesterId: "test-client",
	}

	resp, err := server.GetClusterInfo(context.Background(), req)
	if err != nil {
		t.Fatalf("GetClusterInfo failed: %v", err)
	}

	if resp.ClusterId != "distributed-llm-cluster" {
		t.Errorf("Expected cluster ID 'distributed-llm-cluster', got %s", resp.ClusterId)
	}

	if len(resp.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(resp.Nodes))
	}

	if resp.Metrics == nil {
		t.Error("Expected non-nil metrics")
	}

	if resp.Metrics.TotalNodes != 2 {
		t.Errorf("Expected 2 total nodes, got %d", resp.Metrics.TotalNodes)
	}

	if resp.Metrics.HealthyNodes != 1 {
		t.Errorf("Expected 1 healthy node, got %d", resp.Metrics.HealthyNodes)
	}

	expectedTotalMemory := int64(8192 + 16384)
	if resp.Metrics.TotalMemoryMb != expectedTotalMemory {
		t.Errorf("Expected total memory %d, got %d", expectedTotalMemory, resp.Metrics.TotalMemoryMb)
	}

	expectedUtilization := float32(16) / float32(32+64) * 100.0
	if resp.Metrics.ClusterUtilization != expectedUtilization {
		t.Errorf("Expected utilization %.2f, got %.2f", expectedUtilization, resp.Metrics.ClusterUtilization)
	}
}

func TestTUIServer_GetNodeList(t *testing.T) {
	network, err := NewP2PNetwork("test-node", 8080, 7946)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}
	defer network.Stop()

	discoveryServer := NewDiscoveryServer(network)
	tuiServer := NewTUIServer(network, discoveryServer)

	// Add test nodes
	testNodes := []models.Node{
		{
			ID:      "node-1",
			Address: "localhost",
			Port:    8080,
			Status:  models.NodeStatusOnline,
			Resources: models.ResourceInfo{
				CPUCores: 4,
				MemoryMB: 8192,
			},
			LastSeen: time.Now(),
		},
	}

	network.nodes = testNodes

	req := &pb.NodeListRequest{
		RequesterId:    "tui-client",
		IncludeMetrics: true,
	}

	resp, err := tuiServer.GetNodeList(context.Background(), req)
	if err != nil {
		t.Fatalf("GetNodeList failed: %v", err)
	}

	if len(resp.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(resp.Nodes))
	}

	if resp.Nodes[0].NodeId != "node-1" {
		t.Errorf("Expected node ID 'node-1', got %s", resp.Nodes[0].NodeId)
	}

	if resp.ClusterMetrics == nil {
		t.Error("Expected cluster metrics when IncludeMetrics is true")
	}
}

func TestTUIServer_GetModelList(t *testing.T) {
	network, err := NewP2PNetwork("test-node", 8080, 7946)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}
	defer network.Stop()

	discoveryServer := NewDiscoveryServer(network)
	tuiServer := NewTUIServer(network, discoveryServer)

	req := &pb.ModelListRequest{
		RequesterId: "tui-client",
	}

	resp, err := tuiServer.GetModelList(context.Background(), req)
	if err != nil {
		t.Fatalf("GetModelList failed: %v", err)
	}

	// Should return mock models
	if len(resp.Models) == 0 {
		t.Error("Expected at least one mock model")
	}

	// Check first model
	model := resp.Models[0]
	if model.Id == "" {
		t.Error("Expected non-empty model ID")
	}

	if model.LayerCount == 0 {
		t.Error("Expected non-zero layer count")
	}

	if model.SizeBytes == 0 {
		t.Error("Expected non-zero model size")
	}
}

func TestTUIServer_ExecuteCommand(t *testing.T) {
	network, err := NewP2PNetwork("test-node", 8080, 7946)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}
	defer network.Stop()

	discoveryServer := NewDiscoveryServer(network)
	tuiServer := NewTUIServer(network, discoveryServer)

	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
		expectCode  int32
	}{
		{
			name:        "status command",
			command:     "status",
			args:        []string{},
			expectError: false,
			expectCode:  0,
		},
		{
			name:        "ping command",
			command:     "ping",
			args:        []string{},
			expectError: false,
			expectCode:  0,
		},
		{
			name:        "unknown command",
			command:     "unknown",
			args:        []string{},
			expectError: true,
			expectCode:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.CommandRequest{
				RequesterId: "tui-client",
				Command:     tt.command,
				Args:        tt.args,
			}

			resp, err := tuiServer.ExecuteCommand(context.Background(), req)
			if err != nil {
				t.Fatalf("ExecuteCommand failed: %v", err)
			}

			if resp.Success == tt.expectError {
				t.Errorf("Expected success=%v, got success=%v", !tt.expectError, resp.Success)
			}

			if resp.ExitCode != tt.expectCode {
				t.Errorf("Expected exit code %d, got %d", tt.expectCode, resp.ExitCode)
			}

			if !tt.expectError && resp.Output == "" {
				t.Error("Expected non-empty output for successful command")
			}

			if tt.expectError && resp.Error == "" {
				t.Error("Expected non-empty error for failed command")
			}
		})
	}
}

func TestGRPCServer_Creation(t *testing.T) {
	network, err := NewP2PNetwork("test-node", 8080, 7946)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}
	defer network.Stop()

	// Use a random port for testing
	server, err := NewGRPCServer(network, 0) // 0 means random port
	if err != nil {
		t.Fatalf("Failed to create gRPC server: %v", err)
	}

	if server == nil {
		t.Fatal("Expected non-nil gRPC server")
	}

	if server.nodeServer == nil {
		t.Error("Expected non-nil node server")
	}

	if server.discoveryServer == nil {
		t.Error("Expected non-nil discovery server")
	}

	if server.tuiServer == nil {
		t.Error("Expected non-nil TUI server")
	}

	if server.listener == nil {
		t.Error("Expected non-nil listener")
	}

	address := server.GetAddress()
	if address == "" {
		t.Error("Expected non-empty address")
	}

	// Test graceful stop
	server.Stop()
}

func TestNodeServer_Extended(t *testing.T) {
	network, err := NewP2PNetwork("test-node", 8080, 7946)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}
	defer network.Stop()

	server := &NodeServer{network: network}

	// Test GetNodeInfo
	req := &pb.NodeInfoRequest{
		RequesterId: "test-client",
	}

	resp, err := server.GetNodeInfo(context.Background(), req)
	if err != nil {
		t.Fatalf("GetNodeInfo failed: %v", err)
	}

	if resp.NodeId != "test-node" {
		t.Errorf("Expected node ID 'test-node', got %s", resp.NodeId)
	}

	// Test GetPeers
	peersReq := &pb.PeersRequest{
		RequesterId: "test-client",
	}

	peersResp, err := server.GetPeers(context.Background(), peersReq)
	if err != nil {
		t.Fatalf("GetPeers failed: %v", err)
	}

	if peersResp == nil {
		t.Error("Expected non-nil peers response")
	}

	// Test GetMetrics
	metricsReq := &pb.MetricsRequest{
		RequesterId: "test-client",
	}

	metricsResp, err := server.GetMetrics(context.Background(), metricsReq)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	if metricsResp == nil {
		t.Error("Expected non-nil metrics response")
	}
}
