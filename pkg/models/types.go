package models

import (
	pb "distributed-llm/proto"
	"time"
)

type Node struct {
	ID        string       `json:"id"`
	Address   string       `json:"address"`
	Port      int          `json:"port"`
	Resources ResourceInfo `json:"resources"`
	Status    NodeStatus   `json:"status"`
	LastSeen  time.Time    `json:"last_seen"`
}

type ResourceInfo struct {
	CPUCores   int64     `json:"cpu_cores"`
	MemoryMB   int64     `json:"memory_mb"`
	GPUs       []GPUInfo `json:"gpus"`
	MaxLayers  int32     `json:"max_layers"`
	UsedLayers int32     `json:"used_layers"`
}

type GPUInfo struct {
	Name     string `json:"name"`
	MemoryMB int64  `json:"memory_mb"`
	UUID     string `json:"uuid"`
}

type NodeStatus string

const (
	NodeStatusOnline  NodeStatus = "online"
	NodeStatusOffline NodeStatus = "offline"
	NodeStatusBusy    NodeStatus = "busy"
)

type Model struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	LayerCount int32  `json:"layer_count"`
	FilePath   string `json:"file_path"`
	Size       int64  `json:"size"`
}

type InferenceRequest struct {
	ModelID   string `json:"model_id"`
	Prompt    string `json:"prompt"`
	MaxTokens int32  `json:"max_tokens"`
	RequestID string `json:"request_id"`
}

type InferenceResponse struct {
	RequestID     string `json:"request_id"`
	GeneratedText string `json:"generated_text"`
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
}

type ClusterState struct {
	Nodes  []Node  `json:"nodes"`
	Models []Model `json:"models"`
}

// IsHealthy returns true if the node is online and recently seen
func (n *Node) IsHealthy() bool {
	return n.Status == NodeStatusOnline && time.Since(n.LastSeen) < 5*time.Minute
}

// SizeInGB returns the model size in gigabytes
func (m *Model) SizeInGB() float64 {
	return float64(m.Size) / (1024 * 1024 * 1024)
}

// TotalMemoryMB returns the total memory including CPU memory and GPU memory
func (r *ResourceInfo) TotalMemoryMB() int64 {
	total := r.MemoryMB
	for _, gpu := range r.GPUs {
		total += gpu.MemoryMB
	}
	return total
}

// HasGPUs returns true if the resource has any GPUs
func (r *ResourceInfo) HasGPUs() bool {
	return len(r.GPUs) > 0
}

// Helper functions to convert between protobuf and internal types
func (r *ResourceInfo) ToProto() *pb.ResourceInfo {
	gpus := make([]*pb.GPUInfo, len(r.GPUs))
	for i, gpu := range r.GPUs {
		gpus[i] = &pb.GPUInfo{
			Name:     gpu.Name,
			MemoryMb: gpu.MemoryMB,
			Uuid:     gpu.UUID,
		}
	}

	return &pb.ResourceInfo{
		CpuCores:  r.CPUCores,
		MemoryMb:  r.MemoryMB,
		Gpus:      gpus,
		MaxLayers: r.MaxLayers,
	}
}

func ResourceInfoFromProto(pbRes *pb.ResourceInfo) *ResourceInfo {
	gpus := make([]GPUInfo, len(pbRes.Gpus))
	for i, gpu := range pbRes.Gpus {
		gpus[i] = GPUInfo{
			Name:     gpu.Name,
			MemoryMB: gpu.MemoryMb,
			UUID:     gpu.Uuid,
		}
	}

	return &ResourceInfo{
		CPUCores:  pbRes.CpuCores,
		MemoryMB:  pbRes.MemoryMb,
		GPUs:      gpus,
		MaxLayers: pbRes.MaxLayers,
	}
}
