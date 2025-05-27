package network

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"

	"distributed-llm/pkg/models"
	pb "distributed-llm/proto"
)

// DiscoveryServer implements the gRPC DiscoveryService
type DiscoveryServer struct {
	pb.UnimplementedDiscoveryServiceServer
	network *P2PNetwork
}

func NewDiscoveryServer(network *P2PNetwork) *DiscoveryServer {
	return &DiscoveryServer{network: network}
}

func (d *DiscoveryServer) DiscoverNodes(ctx context.Context, req *pb.DiscoveryRequest) (*pb.DiscoveryResponse, error) {
	nodes := d.network.GetNodes()
	discovered := make([]*pb.NodeInfo, 0, len(nodes))

	for _, node := range nodes {
		// Skip nodes that are already known to the requester
		known := false
		for _, knownNode := range req.KnownNodes {
			if knownNode == node.ID {
				known = true
				break
			}
		}
		if known {
			continue
		}

		gpus := make([]*pb.GPUInfo, len(node.Resources.GPUs))
		for j, gpu := range node.Resources.GPUs {
			gpus[j] = &pb.GPUInfo{
				Name:     gpu.Name,
				MemoryMb: gpu.MemoryMB,
				Uuid:     gpu.UUID,
			}
		}

		discovered = append(discovered, &pb.NodeInfo{
			NodeId:  node.ID,
			Address: node.Address,
			Port:    int32(node.Port),
			Resources: &pb.ResourceInfo{
				CpuCores:   node.Resources.CPUCores,
				MemoryMb:   node.Resources.MemoryMB,
				Gpus:       gpus,
				MaxLayers:  node.Resources.MaxLayers,
				UsedLayers: node.Resources.UsedLayers,
			},
			Status:   string(node.Status),
			LastSeen: node.LastSeen.Unix(),
		})
	}

	return &pb.DiscoveryResponse{
		DiscoveredNodes: discovered,
		Success:         true,
		Message:         fmt.Sprintf("Discovered %d new nodes", len(discovered)),
	}, nil
}

func (d *DiscoveryServer) RegisterWithCluster(ctx context.Context, req *pb.ClusterJoinRequest) (*pb.ClusterJoinResponse, error) {
	d.network.logger.Info("Cluster join request",
		"nodeID", req.NodeId,
		"address", req.Address,
		"port", req.Port,
		"seedNodes", req.SeedNodes)

	// In a real implementation, this would handle cluster join logic
	existingNodes := d.network.GetNodes()
	existingNodeInfos := make([]*pb.NodeInfo, len(existingNodes))

	for i, node := range existingNodes {
		gpus := make([]*pb.GPUInfo, len(node.Resources.GPUs))
		for j, gpu := range node.Resources.GPUs {
			gpus[j] = &pb.GPUInfo{
				Name:     gpu.Name,
				MemoryMb: gpu.MemoryMB,
				Uuid:     gpu.UUID,
			}
		}

		existingNodeInfos[i] = &pb.NodeInfo{
			NodeId:  node.ID,
			Address: node.Address,
			Port:    int32(node.Port),
			Resources: &pb.ResourceInfo{
				CpuCores:   node.Resources.CPUCores,
				MemoryMb:   node.Resources.MemoryMB,
				Gpus:       gpus,
				MaxLayers:  node.Resources.MaxLayers,
				UsedLayers: node.Resources.UsedLayers,
			},
			Status:   string(node.Status),
			LastSeen: node.LastSeen.Unix(),
		}
	}

	return &pb.ClusterJoinResponse{
		Success:       true,
		Message:       "Successfully joined cluster",
		ExistingNodes: existingNodeInfos,
		ClusterId:     "distributed-llm-cluster",
	}, nil
}

func (d *DiscoveryServer) LeaveCluster(ctx context.Context, req *pb.ClusterLeaveRequest) (*pb.ClusterLeaveResponse, error) {
	d.network.logger.Info("Cluster leave request", "nodeID", req.NodeId, "reason", req.Reason)

	// In a real implementation, this would handle cleanup
	return &pb.ClusterLeaveResponse{
		Success: true,
		Message: "Successfully left cluster",
	}, nil
}

func (d *DiscoveryServer) GetClusterInfo(ctx context.Context, req *pb.ClusterInfoRequest) (*pb.ClusterInfoResponse, error) {
	nodes := d.network.GetNodes()
	nodeInfos := make([]*pb.NodeInfo, len(nodes))

	totalCPU := int64(0)
	totalMemory := int64(0)
	totalGPUs := int32(0)
	totalLayers := int32(0)
	allocatedLayers := int32(0)

	for i, node := range nodes {
		gpus := make([]*pb.GPUInfo, len(node.Resources.GPUs))
		for j, gpu := range node.Resources.GPUs {
			gpus[j] = &pb.GPUInfo{
				Name:     gpu.Name,
				MemoryMb: gpu.MemoryMB,
				Uuid:     gpu.UUID,
			}
		}

		nodeInfos[i] = &pb.NodeInfo{
			NodeId:  node.ID,
			Address: node.Address,
			Port:    int32(node.Port),
			Resources: &pb.ResourceInfo{
				CpuCores:   node.Resources.CPUCores,
				MemoryMb:   node.Resources.MemoryMB,
				Gpus:       gpus,
				MaxLayers:  node.Resources.MaxLayers,
				UsedLayers: node.Resources.UsedLayers,
			},
			Status:   string(node.Status),
			LastSeen: node.LastSeen.Unix(),
		}

		// Aggregate metrics
		totalCPU += node.Resources.CPUCores
		totalMemory += node.Resources.MemoryMB
		totalGPUs += int32(len(node.Resources.GPUs))
		totalLayers += node.Resources.MaxLayers
		allocatedLayers += node.Resources.UsedLayers
	}

	healthyNodes := int32(0)
	for _, node := range nodes {
		if node.Status == models.NodeStatusOnline {
			healthyNodes++
		}
	}

	availableMemory := totalMemory // Simplified calculation
	utilization := float32(0.0)
	if totalLayers > 0 {
		utilization = float32(allocatedLayers) / float32(totalLayers) * 100.0
	}

	return &pb.ClusterInfoResponse{
		ClusterId: "distributed-llm-cluster",
		Nodes:     nodeInfos,
		Models:    []*pb.ModelInfo{}, // Empty for now - would be populated from model registry
		Metrics: &pb.ClusterMetrics{
			TotalNodes:         int32(len(nodes)),
			HealthyNodes:       healthyNodes,
			TotalMemoryMb:      totalMemory,
			AvailableMemoryMb:  availableMemory,
			TotalGpus:          totalGPUs,
			TotalLayers:        totalLayers,
			AllocatedLayers:    allocatedLayers,
			ClusterUtilization: utilization,
		},
	}, nil
}

// TUIServer implements the gRPC TUIService
type TUIServer struct {
	pb.UnimplementedTUIServiceServer
	network         *P2PNetwork
	discoveryServer *DiscoveryServer
}

func NewTUIServer(network *P2PNetwork, discoveryServer *DiscoveryServer) *TUIServer {
	return &TUIServer{
		network:         network,
		discoveryServer: discoveryServer,
	}
}

func (t *TUIServer) GetNodeList(ctx context.Context, req *pb.NodeListRequest) (*pb.NodeListResponse, error) {
	nodes := t.network.GetNodes()
	nodeInfos := make([]*pb.NodeInfo, len(nodes))

	for i, node := range nodes {
		gpus := make([]*pb.GPUInfo, len(node.Resources.GPUs))
		for j, gpu := range node.Resources.GPUs {
			gpus[j] = &pb.GPUInfo{
				Name:     gpu.Name,
				MemoryMb: gpu.MemoryMB,
				Uuid:     gpu.UUID,
			}
		}

		nodeInfos[i] = &pb.NodeInfo{
			NodeId:  node.ID,
			Address: node.Address,
			Port:    int32(node.Port),
			Resources: &pb.ResourceInfo{
				CpuCores:   node.Resources.CPUCores,
				MemoryMb:   node.Resources.MemoryMB,
				Gpus:       gpus,
				MaxLayers:  node.Resources.MaxLayers,
				UsedLayers: node.Resources.UsedLayers,
			},
			Status:   string(node.Status),
			LastSeen: node.LastSeen.Unix(),
		}
	}

	// Get cluster metrics if requested
	var clusterMetrics *pb.ClusterMetrics
	if req.IncludeMetrics {
		clusterInfo, err := t.discoveryServer.GetClusterInfo(ctx, &pb.ClusterInfoRequest{
			RequesterId: req.RequesterId,
		})
		if err == nil {
			clusterMetrics = clusterInfo.Metrics
		}
	}

	return &pb.NodeListResponse{
		Nodes:          nodeInfos,
		ClusterMetrics: clusterMetrics,
	}, nil
}

func (t *TUIServer) GetModelList(ctx context.Context, req *pb.ModelListRequest) (*pb.ModelListResponse, error) {
	// Mock models for now - in a real implementation this would come from a model registry
	models := []*pb.ModelInfo{
		{
			Id:              "llama-7b",
			Name:            "Llama 2 7B",
			Version:         "v1.0",
			LayerCount:      32,
			FilePath:        "/models/llama-7b.bin",
			SizeBytes:       13000000000, // 13GB
			NodeAssignments: []string{},
		},
		{
			Id:              "mistral-7b",
			Name:            "Mistral 7B",
			Version:         "v0.1",
			LayerCount:      32,
			FilePath:        "/models/mistral-7b.bin",
			SizeBytes:       14000000000, // 14GB
			NodeAssignments: []string{},
		},
	}

	return &pb.ModelListResponse{
		Models: models,
	}, nil
}

func (t *TUIServer) StreamUpdates(req *pb.UpdateStreamRequest, stream pb.TUIService_StreamUpdatesServer) error {
	interval := time.Duration(req.IntervalSeconds) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second // Default 5 seconds
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			update := &pb.ClusterUpdate{
				Timestamp: time.Now().Unix(),
			}

			// Include requested update types
			for _, updateType := range req.UpdateTypes {
				switch updateType {
				case "nodes":
					nodeResp, err := t.GetNodeList(stream.Context(), &pb.NodeListRequest{
						RequesterId:    req.RequesterId,
						IncludeMetrics: false,
					})
					if err == nil {
						update.UpdateType = "nodes"
						update.Nodes = nodeResp.Nodes
					}

				case "models":
					modelResp, err := t.GetModelList(stream.Context(), &pb.ModelListRequest{
						RequesterId: req.RequesterId,
					})
					if err == nil {
						update.UpdateType = "models"
						update.Models = modelResp.Models
					}

				case "metrics":
					clusterInfo, err := t.discoveryServer.GetClusterInfo(stream.Context(), &pb.ClusterInfoRequest{
						RequesterId: req.RequesterId,
					})
					if err == nil {
						update.UpdateType = "metrics"
						update.Metrics = clusterInfo.Metrics
					}
				}

				if err := stream.Send(update); err != nil {
					return err
				}
			}
		}
	}
}

func (t *TUIServer) ExecuteCommand(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
	t.network.logger.Info("Command execution request",
		"requester", req.RequesterId,
		"command", req.Command,
		"args", req.Args)

	// Mock command execution - in a real implementation this would handle various cluster commands
	switch req.Command {
	case "status":
		return &pb.CommandResponse{
			Success:  true,
			Output:   "Cluster is running normally",
			ExitCode: 0,
		}, nil

	case "ping":
		return &pb.CommandResponse{
			Success:  true,
			Output:   "pong",
			ExitCode: 0,
		}, nil

	default:
		return &pb.CommandResponse{
			Success:  false,
			Error:    fmt.Sprintf("Unknown command: %s", req.Command),
			ExitCode: 1,
		}, nil
	}
}

// GRPCServer wraps the gRPC server with compression support
type GRPCServer struct {
	server          *grpc.Server
	nodeServer      *NodeServer
	discoveryServer *DiscoveryServer
	tuiServer       *TUIServer
	listener        net.Listener
}

func NewGRPCServer(network *P2PNetwork, port int) (*GRPCServer, error) {
	// Create gRPC server with compression enabled
	server := grpc.NewServer(
		grpc.RPCCompressor(grpc.NewGZIPCompressor()),
		grpc.RPCDecompressor(grpc.NewGZIPDecompressor()),
	)

	// Create service implementations
	nodeServer := &NodeServer{network: network}
	discoveryServer := NewDiscoveryServer(network)
	tuiServer := NewTUIServer(network, discoveryServer)

	// Register services
	pb.RegisterNodeServiceServer(server, nodeServer)
	pb.RegisterDiscoveryServiceServer(server, discoveryServer)
	pb.RegisterTUIServiceServer(server, tuiServer)

	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	return &GRPCServer{
		server:          server,
		nodeServer:      nodeServer,
		discoveryServer: discoveryServer,
		tuiServer:       tuiServer,
		listener:        listener,
	}, nil
}

func (g *GRPCServer) Start() error {
	slog.Info("Starting gRPC server with compression", "address", g.listener.Addr().String())
	return g.server.Serve(g.listener)
}

func (g *GRPCServer) Stop() {
	slog.Info("Stopping gRPC server")
	g.server.GracefulStop()
}

func (g *GRPCServer) GetAddress() string {
	return g.listener.Addr().String()
}
