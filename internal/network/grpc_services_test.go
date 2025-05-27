package network

import (
	"context"
	"testing"
	"time"

	"distributed-llm/pkg/models"
	pb "distributed-llm/proto"
)

// MockMetricsCollector implements the MetricsCollector interface for testing
type MockMetricsCollector struct{}

func (m *MockMetricsCollector) RecordNetworkMessage(direction, messageType string) {}
func (m *MockMetricsCollector) RecordNetworkLatency(targetNode, operation string, duration time.Duration) {
}
func (m *MockMetricsCollector) UpdateNodeStatus(status models.NodeStatus) {}
func (m *MockMetricsCollector) UpdateNetworkConnections(count int)        {}
func (m *MockMetricsCollector) RecordInferenceRequest(modelID, status string, duration time.Duration, tokensGenerated int) {
}

func TestDiscoveryServer_DiscoverNodes(t *testing.T) {
	// Create a test P2P network
	network, err := NewP2PNetwork("test-node", 8080, 7946)
	if err != nil {
		t.Fatalf("Failed to create P2P network: %v", err)
	}
	defer network.Stop()

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

	// Since we don't have any actual nodes in memberlist for this test,
	// we should get 0 discovered nodes
	if len(resp.DiscoveredNodes) != 0 {
		t.Errorf("Expected 0 discovered nodes (no memberlist), got %d", len(resp.DiscoveredNodes))
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

	// Since we don't have memberlist started, we should get empty cluster
	if len(resp.Nodes) != 0 {
		t.Errorf("Expected 0 nodes (no memberlist), got %d", len(resp.Nodes))
	}

	if resp.Metrics == nil {
		t.Error("Expected non-nil metrics")
	}

	if resp.Metrics.TotalNodes != 0 {
		t.Errorf("Expected 0 total nodes, got %d", resp.Metrics.TotalNodes)
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

	req := &pb.NodeListRequest{
		RequesterId:    "tui-client",
		IncludeMetrics: true,
	}

	resp, err := tuiServer.GetNodeList(context.Background(), req)
	if err != nil {
		t.Fatalf("GetNodeList failed: %v", err)
	}

	// Since we don't have memberlist started, we should get 0 nodes
	if len(resp.Nodes) != 0 {
		t.Errorf("Expected 0 nodes (no memberlist), got %d", len(resp.Nodes))
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

	// Set up a mock metrics collector
	mockCollector := &MockMetricsCollector{}
	network.SetMetricsCollector(mockCollector)

	server := &NodeServer{network: network}

	// Test GetPeers
	peersReq := &pb.GetPeersRequest{
		NodeId: "test-client",
	}

	peersResp, err := server.GetPeers(context.Background(), peersReq)
	if err != nil {
		t.Fatalf("GetPeers failed: %v", err)
	}

	if peersResp == nil {
		t.Error("Expected non-nil peers response")
	}

	// Test GetMetrics (should work now with mock collector)
	metricsReq := &pb.GetMetricsRequest{
		NodeId: "test-client",
	}

	metricsResp, err := server.GetMetrics(context.Background(), metricsReq)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	if metricsResp == nil {
		t.Error("Expected non-nil metrics response")
	}

	if metricsResp.Metrics == nil {
		t.Error("Expected non-nil metrics")
	}

	// Test HealthCheck
	healthReq := &pb.HealthCheckRequest{
		NodeId: "test-client",
	}

	healthResp, err := server.HealthCheck(context.Background(), healthReq)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if !healthResp.Healthy {
		t.Error("Expected healthy response")
	}

	if healthResp.Status == "" {
		t.Error("Expected non-empty status")
	}
}
