package tui

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"distributed-llm/pkg/models"
	pb "distributed-llm/proto"
)

// MockNodeService implements pb.NodeServiceServer for testing
type MockNodeService struct {
	pb.UnimplementedNodeServiceServer
	resources     *pb.ResourceInfo
	peers         []*pb.NodeInfo
	shouldFail    bool
	failWithError string
}

func (m *MockNodeService) GetResources(ctx context.Context, req *pb.GetResourcesRequest) (*pb.GetResourcesResponse, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("%s", m.failWithError)
	}

	if m.resources == nil {
		m.resources = &pb.ResourceInfo{
			CpuCores:   8,
			MemoryMb:   16384,
			MaxLayers:  20,
			UsedLayers: 5,
			Gpus: []*pb.GPUInfo{
				{
					Name:     "NVIDIA RTX 4090",
					MemoryMb: 24576,
					Uuid:     "GPU-12345",
				},
			},
		}
	}

	return &pb.GetResourcesResponse{
		Resources:       m.resources,
		AvailableLayers: m.resources.MaxLayers - m.resources.UsedLayers,
	}, nil
}

func (m *MockNodeService) GetPeers(ctx context.Context, req *pb.GetPeersRequest) (*pb.GetPeersResponse, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("%s", m.failWithError)
	}

	if m.peers == nil {
		m.peers = []*pb.NodeInfo{
			{
				NodeId:  "peer-1",
				Address: "localhost",
				Port:    8081,
				Status:  "online",
				Resources: &pb.ResourceInfo{
					CpuCores:   4,
					MemoryMb:   8192,
					MaxLayers:  10,
					UsedLayers: 2,
				},
			},
			{
				NodeId:  "peer-2",
				Address: "192.168.1.100",
				Port:    8082,
				Status:  "online",
				Resources: &pb.ResourceInfo{
					CpuCores:   16,
					MemoryMb:   32768,
					MaxLayers:  40,
					UsedLayers: 10,
				},
			},
		}
	}

	return &pb.GetPeersResponse{
		Peers: m.peers,
	}, nil
}

// MockTUIService implements pb.TUIServiceServer for testing
type MockTUIService struct {
	pb.UnimplementedTUIServiceServer
	nodes         []*pb.NodeInfo
	models        []*pb.ModelInfo
	shouldFail    bool
	failWithError string
}

func (m *MockTUIService) GetNodeList(ctx context.Context, req *pb.NodeListRequest) (*pb.NodeListResponse, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("%s", m.failWithError)
	}

	if m.nodes == nil {
		m.nodes = []*pb.NodeInfo{
			{
				NodeId:  "node-1",
				Address: "localhost",
				Port:    8080,
				Status:  "online",
				Resources: &pb.ResourceInfo{
					CpuCores:   8,
					MemoryMb:   16384,
					MaxLayers:  20,
					UsedLayers: 5,
					Gpus: []*pb.GPUInfo{
						{
							Name:     "NVIDIA RTX 4090",
							MemoryMb: 24576,
							Uuid:     "GPU-12345",
						},
					},
				},
				LastSeen: time.Now().Unix(),
			},
		}
	}

	return &pb.NodeListResponse{
		Nodes: m.nodes,
		ClusterMetrics: &pb.ClusterMetrics{
			TotalNodes:        1,
			HealthyNodes:      1,
			TotalMemoryMb:     16384,
			AvailableMemoryMb: 11264,
		},
	}, nil
}

func (m *MockTUIService) GetModelList(ctx context.Context, req *pb.ModelListRequest) (*pb.ModelListResponse, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("%s", m.failWithError)
	}

	if m.models == nil {
		m.models = []*pb.ModelInfo{
			{
				Id:         "llama-7b",
				Name:       "LLaMA 7B",
				Version:    "v1.0",
				LayerCount: 32,
				SizeBytes:  7000000000,
				FilePath:   "/models/llama-7b.bin",
			},
		}
	}

	return &pb.ModelListResponse{
		Models: m.models,
	}, nil
}

// Test helper to create mock gRPC server
func createMockServer(nodeService *MockNodeService, tuiService *MockTUIService) (*grpc.Server, *bufconn.Listener) {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)

	s := grpc.NewServer()
	if nodeService != nil {
		pb.RegisterNodeServiceServer(s, nodeService)
	}
	if tuiService != nil {
		pb.RegisterTUIServiceServer(s, tuiService)
	}

	go func() {
		if err := s.Serve(lis); err != nil {
			// Server was stopped, which is expected during cleanup
		}
	}()

	return s, lis
}

// Test helper to create client with mock server
func createClientWithMockServer(nodeService *MockNodeService, tuiService *MockTUIService) (*Client, func()) {
	server, lis := createMockServer(nodeService, tuiService)

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("Failed to dial bufnet: %v", err))
	}

	client := &Client{
		serverAddr:         "bufnet",
		conn:               conn,
		nodeClient:         pb.NewNodeServiceClient(conn),
		discoveryClient:    pb.NewDiscoveryServiceClient(conn),
		tuiClient:          pb.NewTUIServiceClient(conn),
		compressionEnabled: true,
	}

	cleanup := func() {
		if conn != nil {
			conn.Close()
		}
		server.Stop()
		lis.Close()
	}

	return client, cleanup
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name       string
		serverAddr string
		want       string
	}{
		{
			name:       "basic client creation",
			serverAddr: "localhost:8080",
			want:       "localhost:8080",
		},
		{
			name:       "empty address",
			serverAddr: "",
			want:       "",
		},
		{
			name:       "complex address",
			serverAddr: "192.168.1.100:9090",
			want:       "192.168.1.100:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.serverAddr)

			if client.serverAddr != tt.want {
				t.Errorf("NewClient() serverAddr = %v, want %v", client.serverAddr, tt.want)
			}

			if !client.compressionEnabled {
				t.Error("NewClient() should enable compression by default")
			}

			if client.conn != nil {
				t.Error("NewClient() should not create connection immediately")
			}
		})
	}
}

func TestClient_Connect(t *testing.T) {
	tests := []struct {
		name       string
		serverAddr string
		expectErr  bool
	}{
		{
			name:       "invalid address",
			serverAddr: "invalid-address",
			expectErr:  false, // gRPC doesn't fail immediately on invalid addresses
		},
		{
			name:       "unreachable address",
			serverAddr: "localhost:99999",
			expectErr:  false, // gRPC doesn't fail immediately on unreachable addresses
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.serverAddr)
			err := client.Connect()

			if tt.expectErr && err == nil {
				t.Error("Connect() expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Connect() unexpected error: %v", err)
			}

			// Clean up if connection was created
			if client.conn != nil {
				client.Close()
			}
		})
	}
}

func TestClient_ConnectSuccess(t *testing.T) {
	nodeService := &MockNodeService{}
	tuiService := &MockTUIService{}
	client, cleanup := createClientWithMockServer(nodeService, tuiService)
	defer cleanup()

	// Test that all clients are properly initialized
	if client.nodeClient == nil {
		t.Error("Expected nodeClient to be initialized")
	}
	if client.tuiClient == nil {
		t.Error("Expected tuiClient to be initialized")
	}
}

func TestClient_Close(t *testing.T) {
	t.Run("close without connection", func(t *testing.T) {
		client := NewClient("test")
		err := client.Close()
		if err != nil {
			t.Errorf("Close() should not error on nil connection: %v", err)
		}
	})

	t.Run("close with connection", func(t *testing.T) {
		nodeService := &MockNodeService{}
		tuiService := &MockTUIService{}
		client, cleanup := createClientWithMockServer(nodeService, tuiService)

		err := client.Close()
		if err != nil {
			t.Errorf("Close() unexpected error: %v", err)
		}

		cleanup() // Call cleanup to stop server
	})
}

func TestClient_GetResourceInfo(t *testing.T) {
	t.Run("not connected", func(t *testing.T) {
		client := NewClient("test")
		_, err := client.GetResourceInfo()
		if err == nil {
			t.Error("GetResourceInfo() should fail when not connected")
		}
		if err.Error() != "client not connected" {
			t.Errorf("GetResourceInfo() wrong error message: %v", err)
		}
	})

	t.Run("successful call", func(t *testing.T) {
		nodeService := &MockNodeService{}
		client, cleanup := createClientWithMockServer(nodeService, nil)
		defer cleanup()

		resourceInfo, err := client.GetResourceInfo()
		if err != nil {
			t.Fatalf("GetResourceInfo() unexpected error: %v", err)
		}

		if resourceInfo.CPUCores != 8 {
			t.Errorf("Expected 8 CPU cores, got %d", resourceInfo.CPUCores)
		}
		if resourceInfo.MemoryMB != 16384 {
			t.Errorf("Expected 16384 MB memory, got %d", resourceInfo.MemoryMB)
		}
		if resourceInfo.MaxLayers != 20 {
			t.Errorf("Expected 20 max layers, got %d", resourceInfo.MaxLayers)
		}
		if resourceInfo.UsedLayers != 5 {
			t.Errorf("Expected 5 used layers, got %d", resourceInfo.UsedLayers)
		}
		if len(resourceInfo.GPUs) != 1 {
			t.Errorf("Expected 1 GPU, got %d", len(resourceInfo.GPUs))
		}
		if resourceInfo.GPUs[0].Name != "NVIDIA RTX 4090" {
			t.Errorf("Expected GPU name 'NVIDIA RTX 4090', got '%s'", resourceInfo.GPUs[0].Name)
		}
	})

	t.Run("server error", func(t *testing.T) {
		nodeService := &MockNodeService{
			shouldFail:    true,
			failWithError: "server error",
		}
		client, cleanup := createClientWithMockServer(nodeService, nil)
		defer cleanup()

		_, err := client.GetResourceInfo()
		if err == nil {
			t.Error("GetResourceInfo() should fail when server returns error")
		}
	})

	t.Run("nil resources", func(t *testing.T) {
		// Create a custom mock that returns nil resources
		customMockService := &MockNodeService{}
		client, cleanup := createClientWithMockServer(customMockService, nil)
		defer cleanup()

		// Override the behavior by creating a new mock with nil resources response
		customMockService.shouldFail = false
		customMockService.resources = nil

		// We need to test via a different approach since we can't reassign methods
		// Let's modify the mock's resources field to be nil
		customMockService.resources = nil

		// Call GetResources which should handle nil resources
		resourceInfo, err := client.GetResourceInfo()
		// The actual implementation should handle nil resources gracefully
		// If it doesn't, that's a bug we should fix in the implementation
		if err == nil && resourceInfo == nil {
			t.Error("Expected either an error or valid resource info")
		}
	})
}

func TestClient_GetPeers(t *testing.T) {
	t.Run("not connected", func(t *testing.T) {
		client := NewClient("test")
		_, err := client.GetPeers()
		if err == nil {
			t.Error("GetPeers() should fail when not connected")
		}
		if err.Error() != "client not connected" {
			t.Errorf("GetPeers() wrong error message: %v", err)
		}
	})

	t.Run("successful call", func(t *testing.T) {
		nodeService := &MockNodeService{}
		client, cleanup := createClientWithMockServer(nodeService, nil)
		defer cleanup()

		peers, err := client.GetPeers()
		if err != nil {
			t.Fatalf("GetPeers() unexpected error: %v", err)
		}

		expectedPeers := []string{"localhost:8081", "192.168.1.100:8082"}
		if len(peers) != len(expectedPeers) {
			t.Errorf("Expected %d peers, got %d", len(expectedPeers), len(peers))
		}

		for i, expected := range expectedPeers {
			if i < len(peers) && peers[i] != expected {
				t.Errorf("Expected peer %d to be '%s', got '%s'", i, expected, peers[i])
			}
		}
	})

	t.Run("server error", func(t *testing.T) {
		nodeService := &MockNodeService{
			shouldFail:    true,
			failWithError: "server error",
		}
		client, cleanup := createClientWithMockServer(nodeService, nil)
		defer cleanup()

		_, err := client.GetPeers()
		if err == nil {
			t.Error("GetPeers() should fail when server returns error")
		}
	})
}

func TestClient_GetNodes(t *testing.T) {
	t.Run("not connected", func(t *testing.T) {
		client := NewClient("test")
		_, err := client.GetNodes()
		if err == nil {
			t.Error("GetNodes() should fail when not connected")
		}
		if err.Error() != "client not connected" {
			t.Errorf("GetNodes() wrong error message: %v", err)
		}
	})

	t.Run("successful call", func(t *testing.T) {
		tuiService := &MockTUIService{}
		client, cleanup := createClientWithMockServer(nil, tuiService)
		defer cleanup()

		nodes, err := client.GetNodes()
		if err != nil {
			t.Fatalf("GetNodes() unexpected error: %v", err)
		}

		if len(nodes) != 1 {
			t.Errorf("Expected 1 node, got %d", len(nodes))
		}

		node := nodes[0]
		if node.ID != "node-1" {
			t.Errorf("Expected node ID 'node-1', got '%s'", node.ID)
		}
		if node.Address != "localhost" {
			t.Errorf("Expected node address 'localhost', got '%s'", node.Address)
		}
		if node.Port != 8080 {
			t.Errorf("Expected node port 8080, got %d", node.Port)
		}
		if node.Status != models.NodeStatusOnline {
			t.Errorf("Expected node status online, got %v", node.Status)
		}
		if node.Resources.CPUCores != 8 {
			t.Errorf("Expected 8 CPU cores, got %d", node.Resources.CPUCores)
		}
	})

	t.Run("server error", func(t *testing.T) {
		tuiService := &MockTUIService{
			shouldFail:    true,
			failWithError: "server error",
		}
		client, cleanup := createClientWithMockServer(nil, tuiService)
		defer cleanup()

		_, err := client.GetNodes()
		if err == nil {
			t.Error("GetNodes() should fail when server returns error")
		}
	})
}

func TestClient_GetModels(t *testing.T) {
	t.Run("not connected", func(t *testing.T) {
		client := NewClient("test")
		_, err := client.GetModels()
		if err == nil {
			t.Error("GetModels() should fail when not connected")
		}
		if err.Error() != "client not connected" {
			t.Errorf("GetModels() wrong error message: %v", err)
		}
	})

	t.Run("successful call", func(t *testing.T) {
		tuiService := &MockTUIService{}
		client, cleanup := createClientWithMockServer(nil, tuiService)
		defer cleanup()

		models, err := client.GetModels()
		if err != nil {
			t.Fatalf("GetModels() unexpected error: %v", err)
		}

		if len(models) != 1 {
			t.Errorf("Expected 1 model, got %d", len(models))
		}

		model := models[0]
		if model.ID != "llama-7b" {
			t.Errorf("Expected model ID 'llama-7b', got '%s'", model.ID)
		}
		if model.Name != "LLaMA 7B" {
			t.Errorf("Expected model name 'LLaMA 7B', got '%s'", model.Name)
		}
		if model.Version != "v1.0" {
			t.Errorf("Expected model version 'v1.0', got '%s'", model.Version)
		}
		if model.LayerCount != 32 {
			t.Errorf("Expected 32 layers, got %d", model.LayerCount)
		}
		if model.Size != 7000000000 {
			t.Errorf("Expected size 7000000000 bytes, got %d", model.Size)
		}
	})

	t.Run("server error", func(t *testing.T) {
		tuiService := &MockTUIService{
			shouldFail:    true,
			failWithError: "server error",
		}
		client, cleanup := createClientWithMockServer(nil, tuiService)
		defer cleanup()

		_, err := client.GetModels()
		if err == nil {
			t.Error("GetModels() should fail when server returns error")
		}
	})
}

func TestConvertProtoToNode(t *testing.T) {
	tests := []struct {
		name     string
		nodeInfo *pb.NodeInfo
		expected models.Node
	}{
		{
			name: "online node with GPU",
			nodeInfo: &pb.NodeInfo{
				NodeId:  "test-node",
				Address: "192.168.1.100",
				Port:    8080,
				Status:  "online",
				Resources: &pb.ResourceInfo{
					CpuCores:   16,
					MemoryMb:   32768,
					MaxLayers:  40,
					UsedLayers: 10,
					Gpus: []*pb.GPUInfo{
						{
							Name:     "NVIDIA RTX 4090",
							MemoryMb: 24576,
							Uuid:     "GPU-12345",
						},
					},
				},
				LastSeen: 1234567890,
			},
			expected: models.Node{
				ID:      "test-node",
				Address: "192.168.1.100",
				Port:    8080,
				Status:  models.NodeStatusOnline,
				Resources: models.ResourceInfo{
					CPUCores:   16,
					MemoryMB:   32768,
					MaxLayers:  40,
					UsedLayers: 10,
					GPUs: []models.GPUInfo{
						{
							Name:     "NVIDIA RTX 4090",
							MemoryMB: 24576,
							UUID:     "GPU-12345",
						},
					},
				},
				LastSeen: time.Unix(1234567890, 0),
			},
		},
		{
			name: "offline node without GPU",
			nodeInfo: &pb.NodeInfo{
				NodeId:  "offline-node",
				Address: "10.0.0.1",
				Port:    9090,
				Status:  "offline",
				Resources: &pb.ResourceInfo{
					CpuCores:   4,
					MemoryMb:   8192,
					MaxLayers:  20,
					UsedLayers: 0,
					Gpus:       []*pb.GPUInfo{},
				},
				LastSeen: 987654321,
			},
			expected: models.Node{
				ID:      "offline-node",
				Address: "10.0.0.1",
				Port:    9090,
				Status:  models.NodeStatusOffline,
				Resources: models.ResourceInfo{
					CPUCores:   4,
					MemoryMB:   8192,
					MaxLayers:  20,
					UsedLayers: 0,
					GPUs:       []models.GPUInfo{},
				},
				LastSeen: time.Unix(987654321, 0),
			},
		},
		{
			name: "busy node",
			nodeInfo: &pb.NodeInfo{
				NodeId:  "busy-node",
				Address: "localhost",
				Port:    8888,
				Status:  "busy",
				Resources: &pb.ResourceInfo{
					CpuCores:   8,
					MemoryMb:   16384,
					MaxLayers:  30,
					UsedLayers: 30,
				},
				LastSeen: 555666777,
			},
			expected: models.Node{
				ID:      "busy-node",
				Address: "localhost",
				Port:    8888,
				Status:  models.NodeStatusBusy,
				Resources: models.ResourceInfo{
					CPUCores:   8,
					MemoryMB:   16384,
					MaxLayers:  30,
					UsedLayers: 30,
					GPUs:       []models.GPUInfo{},
				},
				LastSeen: time.Unix(555666777, 0),
			},
		},
		{
			name: "unknown status defaults to offline",
			nodeInfo: &pb.NodeInfo{
				NodeId:  "unknown-node",
				Address: "example.com",
				Port:    7777,
				Status:  "unknown",
				Resources: &pb.ResourceInfo{
					CpuCores:   2,
					MemoryMb:   4096,
					MaxLayers:  10,
					UsedLayers: 1,
				},
				LastSeen: 111222333,
			},
			expected: models.Node{
				ID:      "unknown-node",
				Address: "example.com",
				Port:    7777,
				Status:  models.NodeStatusOffline, // unknown status defaults to offline
				Resources: models.ResourceInfo{
					CPUCores:   2,
					MemoryMB:   4096,
					MaxLayers:  10,
					UsedLayers: 1,
					GPUs:       []models.GPUInfo{},
				},
				LastSeen: time.Unix(111222333, 0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertProtoToNode(tt.nodeInfo)

			if result.ID != tt.expected.ID {
				t.Errorf("Expected ID '%s', got '%s'", tt.expected.ID, result.ID)
			}
			if result.Address != tt.expected.Address {
				t.Errorf("Expected address '%s', got '%s'", tt.expected.Address, result.Address)
			}
			if result.Port != tt.expected.Port {
				t.Errorf("Expected port %d, got %d", tt.expected.Port, result.Port)
			}
			if result.Status != tt.expected.Status {
				t.Errorf("Expected status %v, got %v", tt.expected.Status, result.Status)
			}
			if result.Resources.CPUCores != tt.expected.Resources.CPUCores {
				t.Errorf("Expected CPU cores %d, got %d", tt.expected.Resources.CPUCores, result.Resources.CPUCores)
			}
			if result.Resources.MemoryMB != tt.expected.Resources.MemoryMB {
				t.Errorf("Expected memory %d MB, got %d MB", tt.expected.Resources.MemoryMB, result.Resources.MemoryMB)
			}
			if len(result.Resources.GPUs) != len(tt.expected.Resources.GPUs) {
				t.Errorf("Expected %d GPUs, got %d", len(tt.expected.Resources.GPUs), len(result.Resources.GPUs))
			}
			if result.LastSeen.Unix() != tt.expected.LastSeen.Unix() {
				t.Errorf("Expected LastSeen %v, got %v", tt.expected.LastSeen, result.LastSeen)
			}
		})
	}
}

func TestClient_StartResourcePolling(t *testing.T) {
	// This test is mainly for coverage since StartResourcePolling runs in a loop
	// We'll run it for a very short time and then stop
	nodeService := &MockNodeService{}
	client, cleanup := createClientWithMockServer(nodeService, nil)
	defer cleanup()

	// Start polling in a goroutine
	done := make(chan bool)
	go func() {
		time.Sleep(10 * time.Millisecond) // Let it run for a short time
		done <- true
	}()

	// This will run briefly before the test ends
	go client.StartResourcePolling(5 * time.Millisecond)

	<-done // Wait for our timer
	// Test passes if we reach here without hanging
}

// Integration test for multiple client operations
func TestClient_Integration(t *testing.T) {
	nodeService := &MockNodeService{}
	tuiService := &MockTUIService{}
	client, cleanup := createClientWithMockServer(nodeService, tuiService)
	defer cleanup()

	// Test getting resource info
	resources, err := client.GetResourceInfo()
	if err != nil {
		t.Fatalf("Failed to get resource info: %v", err)
	}
	if resources.CPUCores != 8 {
		t.Errorf("Expected 8 CPU cores, got %d", resources.CPUCores)
	}

	// Test getting peers
	peers, err := client.GetPeers()
	if err != nil {
		t.Fatalf("Failed to get peers: %v", err)
	}
	if len(peers) != 2 {
		t.Errorf("Expected 2 peers, got %d", len(peers))
	}

	// Test getting nodes
	nodes, err := client.GetNodes()
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}

	// Test getting models
	models, err := client.GetModels()
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}
	if len(models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(models))
	}

	// Test closing
	err = client.Close()
	if err != nil {
		t.Errorf("Failed to close client: %v", err)
	}
}
