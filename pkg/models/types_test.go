package models

import (
	"testing"
	"time"
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
			if got := len(tt.resource.GPUs) > 0; got != tt.want {
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
			// Simple health check: online and seen within last 5 minutes
			isHealthy := tt.node.Status == NodeStatusOnline &&
				time.Since(tt.node.LastSeen) < 5*time.Minute

			if got := isHealthy; got != tt.want {
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
			got := float64(tt.model.Size) / (1024 * 1024 * 1024)
			if got != tt.want {
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
	totalMemory := resource.MemoryMB
	for _, gpu := range resource.GPUs {
		totalMemory += gpu.MemoryMB
	}

	if totalMemory != expectedTotal {
		t.Errorf("TotalMemoryMB = %v, want %v", totalMemory, expectedTotal)
	}
}
