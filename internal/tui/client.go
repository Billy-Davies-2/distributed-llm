package tui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"

	"distributed-llm/pkg/models"
	pb "distributed-llm/proto"
)

type Client struct {
	serverAddr         string
	conn               *grpc.ClientConn
	nodeClient         pb.NodeServiceClient
	discoveryClient    pb.DiscoveryServiceClient
	tuiClient          pb.TUIServiceClient
	compressionEnabled bool
}

func NewClient(serverAddr string) *Client {
	return &Client{
		serverAddr:         serverAddr,
		compressionEnabled: true, // Enable compression by default
	}
}

func (c *Client) Connect() error {
	var err error

	// Configure gRPC dial options with compression
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Add compression if enabled
	if c.compressionEnabled {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
	}

	c.conn, err = grpc.NewClient(c.serverAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.nodeClient = pb.NewNodeServiceClient(c.conn)
	c.discoveryClient = pb.NewDiscoveryServiceClient(c.conn)
	c.tuiClient = pb.NewTUIServiceClient(c.conn)
	return nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) GetResourceInfo() (*models.ResourceInfo, error) {
	if c.nodeClient == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.nodeClient.GetResources(ctx, &pb.GetResourcesRequest{NodeId: "tui-client"})
	if err != nil {
		return nil, fmt.Errorf("failed to get resources: %w", err)
	}

	// Convert protobuf response to models.ResourceInfo
	if resp.Resources == nil {
		return nil, fmt.Errorf("no resource info returned")
	}

	gpus := make([]models.GPUInfo, len(resp.Resources.Gpus))
	for i, gpu := range resp.Resources.Gpus {
		gpus[i] = models.GPUInfo{
			Name:     gpu.Name,
			MemoryMB: gpu.MemoryMb,
			UUID:     gpu.Uuid,
		}
	}

	return &models.ResourceInfo{
		CPUCores:   resp.Resources.CpuCores,
		MemoryMB:   resp.Resources.MemoryMb,
		GPUs:       gpus,
		MaxLayers:  resp.Resources.MaxLayers,
		UsedLayers: resp.Resources.UsedLayers,
	}, nil
}

// GetPeers gets peer list from the agent using the new protobuf method
func (c *Client) GetPeers() ([]string, error) {
	if c.nodeClient == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.nodeClient.GetPeers(ctx, &pb.GetPeersRequest{NodeId: "tui-client"})
	if err != nil {
		return nil, fmt.Errorf("failed to get peers: %w", err)
	}

	peers := make([]string, len(resp.Peers))
	for i, peer := range resp.Peers {
		peers[i] = fmt.Sprintf("%s:%d", peer.Address, peer.Port)
	}

	return peers, nil
}

// GetNodes gets all nodes in the cluster using TUI service
func (c *Client) GetNodes() ([]models.Node, error) {
	if c.tuiClient == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.tuiClient.GetNodeList(ctx, &pb.NodeListRequest{
		RequesterId:    "tui-client",
		IncludeMetrics: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get node list: %w", err)
	}

	nodes := make([]models.Node, len(resp.Nodes))
	for i, nodeInfo := range resp.Nodes {
		nodes[i] = convertProtoToNode(nodeInfo)
	}

	return nodes, nil
}

// GetModels gets all models in the cluster using TUI service
func (c *Client) GetModels() ([]models.Model, error) {
	if c.tuiClient == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.tuiClient.GetModelList(ctx, &pb.ModelListRequest{RequesterId: "tui-client"})
	if err != nil {
		return nil, fmt.Errorf("failed to get model list: %w", err)
	}

	modelList := make([]models.Model, len(resp.Models))
	for i, modelInfo := range resp.Models {
		modelList[i] = models.Model{
			ID:         modelInfo.Id,
			Name:       modelInfo.Name,
			Version:    modelInfo.Version,
			LayerCount: modelInfo.LayerCount,
			FilePath:   modelInfo.FilePath,
			Size:       modelInfo.SizeBytes,
		}
	}

	return modelList, nil
}

// convertProtoToNode converts protobuf NodeInfo to models.Node
func convertProtoToNode(nodeInfo *pb.NodeInfo) models.Node {
	gpus := make([]models.GPUInfo, len(nodeInfo.Resources.Gpus))
	for i, gpu := range nodeInfo.Resources.Gpus {
		gpus[i] = models.GPUInfo{
			Name:     gpu.Name,
			MemoryMB: gpu.MemoryMb,
			UUID:     gpu.Uuid,
		}
	}

	var status models.NodeStatus
	switch nodeInfo.Status {
	case "online":
		status = models.NodeStatusOnline
	case "offline":
		status = models.NodeStatusOffline
	case "busy":
		status = models.NodeStatusBusy
	default:
		status = models.NodeStatusOffline
	}

	return models.Node{
		ID:      nodeInfo.NodeId,
		Address: nodeInfo.Address,
		Port:    int(nodeInfo.Port),
		Status:  status,
		Resources: models.ResourceInfo{
			CPUCores:   nodeInfo.Resources.CpuCores,
			MemoryMB:   nodeInfo.Resources.MemoryMb,
			GPUs:       gpus,
			MaxLayers:  nodeInfo.Resources.MaxLayers,
			UsedLayers: nodeInfo.Resources.UsedLayers,
		},
		LastSeen: time.Unix(nodeInfo.LastSeen, 0),
	}
}

func (c *Client) StartResourcePolling(interval time.Duration) {
	slog.Info("Starting retro resource polling", "interval", interval)

	for {
		resourceInfo, err := c.GetResourceInfo()
		if err != nil {
			slog.Error("Failed to fetch resource info from node", "error", err)
			time.Sleep(interval)
			continue
		}

		// Retro-style terminal output
		fmt.Printf("\n▓▓▓ RESOURCE UPDATE ▓▓▓\n")
		fmt.Printf("CPU: %d CORES │ RAM: %d MB\n", resourceInfo.CPUCores, resourceInfo.MemoryMB)
		if len(resourceInfo.GPUs) > 0 {
			fmt.Printf("GPU: ")
			for i, gpu := range resourceInfo.GPUs {
				if i > 0 {
					fmt.Printf(" │ ")
				}
				fmt.Printf("%s (%dMB)", gpu.Name, gpu.MemoryMB)
			}
			fmt.Printf("\n")
		}
		fmt.Printf("LAYERS: %d/%d AVAILABLE\n", resourceInfo.MaxLayers-resourceInfo.UsedLayers, resourceInfo.MaxLayers)

		time.Sleep(interval)
	}
}
