package models

import (
	"testing"
	"time"

	pb "distributed-llm/proto"
)

func TestNodeStatus(t *testing.T) {
	tests := []struct {
		name   string
		status NodeStatus
		want   string
	}{
		{"Online status", NodeStatusOnline, "online"},
		{"Offline status", NodeStatusOffline, "offline"},
		{"Busy status", NodeStatusBusy, "busy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.status); got != tt.want {
				t.Errorf("NodeStatus = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceInfo_HasGPUs(t *testing.T) {
	tests := []struct {
		name     string
		resource ResourceInfo
		want     bool
	}{
		{
			name: "No GPUs",
			resource: ResourceInfo{
				CPUCores: 4,
				MemoryMB: 8192,
				GPUs:     []GPUInfo{},
			},
			want: false,
		},
		{
			name: "Has GPUs",
			resource: ResourceInfo{
				CPUCores: 8,
				MemoryMB: 16384,
				GPUs: []GPUInfo{
					{Name: "RTX 4090", MemoryMB: 24576},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resource.HasGPUs(); got != tt.want {
				t.Errorf("ResourceInfo.HasGPUs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNode_IsHealthy(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		node Node
		want bool
	}{
		{
			name: "Healthy online node",
			node: Node{
				ID:        "node1",
				Status:    NodeStatusOnline,
				LastSeen:  now,
				Resources: ResourceInfo{CPUCores: 4, MemoryMB: 8192},
			},
			want: true,
		},
		{
			name: "Offline node",
			node: Node{
				ID:        "node2",
				Status:    NodeStatusOffline,
				LastSeen:  now.Add(-5 * time.Minute),
				Resources: ResourceInfo{CPUCores: 4, MemoryMB: 8192},
			},
			want: false,
		},
		{
			name: "Stale node",
			node: Node{
				ID:        "node3",
				Status:    NodeStatusOnline,
				LastSeen:  now.Add(-10 * time.Minute),
				Resources: ResourceInfo{CPUCores: 4, MemoryMB: 8192},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.IsHealthy(); got != tt.want {
				t.Errorf("Node.IsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModel_SizeInGB(t *testing.T) {
	tests := []struct {
		name  string
		model Model
		want  float64
	}{
		{
			name: "Small model",
			model: Model{
				ID:   "model1",
				Size: 1024 * 1024 * 1024, // 1GB
			},
			want: 1.0,
		},
		{
			name: "Large model",
			model: Model{
				ID:   "model2",
				Size: 7 * 1024 * 1024 * 1024, // 7GB
			},
			want: 7.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.SizeInGB(); got != tt.want {
				t.Errorf("Model.SizeInGB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceInfo_TotalMemoryMB(t *testing.T) {
	resource := ResourceInfo{
		CPUCores: 8,
		MemoryMB: 16384,
		GPUs: []GPUInfo{
			{Name: "RTX 4090", MemoryMB: 24576},
			{Name: "RTX 4080", MemoryMB: 16384},
		},
	}

	expectedTotal := int64(16384 + 24576 + 16384) // RAM + GPU1 + GPU2
	totalMemory := resource.TotalMemoryMB()

	if totalMemory != expectedTotal {
		t.Errorf("TotalMemoryMB() = %v, want %v", totalMemory, expectedTotal)
	}
}

func TestResourceInfo_ToProto(t *testing.T) {
	resource := ResourceInfo{
		CPUCores: 8,
		MemoryMB: 16384,
		GPUs: []GPUInfo{
			{Name: "RTX 4090", MemoryMB: 24576, UUID: "gpu-1"},
			{Name: "RTX 4080", MemoryMB: 16384, UUID: "gpu-2"},
		},
		MaxLayers:  32,
		UsedLayers: 16,
	}

	proto := resource.ToProto()

	if proto.CpuCores != 8 {
		t.Errorf("Expected CpuCores 8, got %d", proto.CpuCores)
	}

	if proto.MemoryMb != 16384 {
		t.Errorf("Expected MemoryMb 16384, got %d", proto.MemoryMb)
	}

	if proto.MaxLayers != 32 {
		t.Errorf("Expected MaxLayers 32, got %d", proto.MaxLayers)
	}

	if len(proto.Gpus) != 2 {
		t.Errorf("Expected 2 GPUs, got %d", len(proto.Gpus))
	}

	if proto.Gpus[0].Name != "RTX 4090" {
		t.Errorf("Expected GPU name RTX 4090, got %s", proto.Gpus[0].Name)
	}

	if proto.Gpus[0].MemoryMb != 24576 {
		t.Errorf("Expected GPU memory 24576, got %d", proto.Gpus[0].MemoryMb)
	}

	if proto.Gpus[0].Uuid != "gpu-1" {
		t.Errorf("Expected GPU UUID gpu-1, got %s", proto.Gpus[0].Uuid)
	}
}

func TestResourceInfoFromProto(t *testing.T) {
	proto := &pb.ResourceInfo{
		CpuCores: 8,
		MemoryMb: 16384,
		Gpus: []*pb.GPUInfo{
			{Name: "RTX 4090", MemoryMb: 24576, Uuid: "gpu-1"},
			{Name: "RTX 4080", MemoryMb: 16384, Uuid: "gpu-2"},
		},
		MaxLayers: 32,
	}

	resource := ResourceInfoFromProto(proto)

	if resource.CPUCores != 8 {
		t.Errorf("Expected CPUCores 8, got %d", resource.CPUCores)
	}

	if resource.MemoryMB != 16384 {
		t.Errorf("Expected MemoryMB 16384, got %d", resource.MemoryMB)
	}

	if resource.MaxLayers != 32 {
		t.Errorf("Expected MaxLayers 32, got %d", resource.MaxLayers)
	}

	if len(resource.GPUs) != 2 {
		t.Errorf("Expected 2 GPUs, got %d", len(resource.GPUs))
	}

	if resource.GPUs[0].Name != "RTX 4090" {
		t.Errorf("Expected GPU name RTX 4090, got %s", resource.GPUs[0].Name)
	}

	if resource.GPUs[0].MemoryMB != 24576 {
		t.Errorf("Expected GPU memory 24576, got %d", resource.GPUs[0].MemoryMB)
	}

	if resource.GPUs[0].UUID != "gpu-1" {
		t.Errorf("Expected GPU UUID gpu-1, got %s", resource.GPUs[0].UUID)
	}
}

func TestInferenceTypes(t *testing.T) {
	// Test InferenceRequest
	req := InferenceRequest{
		ModelID:   "llama-7b",
		Prompt:    "Hello, world!",
		MaxTokens: 100,
		RequestID: "req-123",
	}

	if req.ModelID != "llama-7b" {
		t.Errorf("Expected ModelID llama-7b, got %s", req.ModelID)
	}

	if req.Prompt != "Hello, world!" {
		t.Errorf("Expected Prompt 'Hello, world!', got %s", req.Prompt)
	}

	// Test InferenceResponse
	resp := InferenceResponse{
		RequestID:     "req-123",
		GeneratedText: "Hello, world! How are you?",
		Success:       true,
		Error:         "",
	}

	if resp.RequestID != "req-123" {
		t.Errorf("Expected RequestID req-123, got %s", resp.RequestID)
	}

	if !resp.Success {
		t.Error("Expected Success to be true")
	}

	if resp.Error != "" {
		t.Errorf("Expected empty Error, got %s", resp.Error)
	}
}

func TestClusterState(t *testing.T) {
	nodes := []Node{
		{
			ID:      "node-1",
			Status:  NodeStatusOnline,
			Address: "localhost",
			Port:    8080,
		},
		{
			ID:      "node-2",
			Status:  NodeStatusBusy,
			Address: "localhost",
			Port:    8081,
		},
	}

	models := []Model{
		{
			ID:         "llama-7b",
			Name:       "Llama 2 7B",
			Version:    "v1.0",
			LayerCount: 32,
			Size:       7 * 1024 * 1024 * 1024, // 7GB
		},
	}

	state := ClusterState{
		Nodes:  nodes,
		Models: models,
	}

	if len(state.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(state.Nodes))
	}

	if len(state.Models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(state.Models))
	}

	if state.Nodes[0].Status != NodeStatusOnline {
		t.Errorf("Expected first node to be online, got %s", state.Nodes[0].Status)
	}

	if state.Models[0].SizeInGB() != 7.0 {
		t.Errorf("Expected model size 7GB, got %f", state.Models[0].SizeInGB())
	}
}
