package network

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hashicorp/memberlist"

	"distributed-llm/pkg/models"
	pb "distributed-llm/proto"
)

type P2PNetwork struct {
	memberlist    *memberlist.Memberlist
	nodeID        string
	bindPort      int
	gossipPort    int
	logger        *slog.Logger
	eventDelegate *EventDelegate
}

type EventDelegate struct {
	network *P2PNetwork
	logger  *slog.Logger
}

func (e *EventDelegate) NotifyJoin(node *memberlist.Node) {
	e.logger.Info("Node joined", "name", node.Name, "addr", node.Addr)
}

func (e *EventDelegate) NotifyLeave(node *memberlist.Node) {
	e.logger.Info("Node left", "name", node.Name, "addr", node.Addr)
}

func (e *EventDelegate) NotifyUpdate(node *memberlist.Node) {
	e.logger.Info("Node updated", "name", node.Name, "addr", node.Addr)
}

func NewP2PNetwork(nodeID string, bindPort, gossipPort int) (*P2PNetwork, error) {
	logger := slog.Default()

	// Validate node ID
	if nodeID == "" {
		return nil, fmt.Errorf("node ID cannot be empty")
	}
	if strings.ContainsAny(nodeID, "\n\r\t") {
		return nil, fmt.Errorf("node ID cannot contain newlines or tabs")
	}

	// Validate ports
	if bindPort <= 0 || bindPort > 65535 {
		return nil, fmt.Errorf("invalid bind port: %d", bindPort)
	}
	if gossipPort <= 0 || gossipPort > 65535 {
		return nil, fmt.Errorf("invalid gossip port: %d", gossipPort)
	}
	if bindPort == gossipPort {
		return nil, fmt.Errorf("bind port and gossip port cannot be the same: %d", bindPort)
	}

	network := &P2PNetwork{
		nodeID:     nodeID,
		bindPort:   bindPort,
		gossipPort: gossipPort,
		logger:     logger,
	}

	network.eventDelegate = &EventDelegate{
		network: network,
		logger:  logger,
	}

	return network, nil
}

func (n *P2PNetwork) Start(seedNodes []string) error {
	// Configure memberlist
	config := memberlist.DefaultLocalConfig()
	config.Name = n.nodeID
	config.BindPort = n.gossipPort
	config.AdvertisePort = n.gossipPort
	config.Events = n.eventDelegate

	// Create memberlist
	list, err := memberlist.Create(config)
	if err != nil {
		return fmt.Errorf("failed to create memberlist: %w", err)
	}
	n.memberlist = list

	// Join existing cluster if seed nodes provided
	if len(seedNodes) > 0 {
		_, err := list.Join(seedNodes)
		if err != nil {
			n.logger.Warn("Failed to join cluster", "error", err)
		} else {
			n.logger.Info("Joined cluster with seeds", "seeds", seedNodes)
		}
	}

	n.logger.Info("P2P network started", "port", n.gossipPort)
	return nil
}

func (n *P2PNetwork) GetNodes() []models.Node {
	nodes := make([]models.Node, 0)

	if n.memberlist == nil {
		return nodes
	}

	for _, member := range n.memberlist.Members() {
		node := models.Node{
			ID:       member.Name,
			Address:  member.Addr.String(),
			Port:     int(member.Port),
			Status:   models.NodeStatusOnline,
			LastSeen: time.Now(),
		}
		nodes = append(nodes, node)
	}

	return nodes
}

func (n *P2PNetwork) GetMembers() []string {
	members := make([]string, 0)

	if n.memberlist == nil {
		return members
	}

	for _, member := range n.memberlist.Members() {
		members = append(members, member.Name)
	}

	return members
}

func (n *P2PNetwork) Stop() {
	if n.memberlist != nil {
		n.memberlist.Shutdown()
	}
}

// NodeServer implements the gRPC NodeService
type NodeServer struct {
	pb.UnimplementedNodeServiceServer
	network *P2PNetwork
}

func (s *NodeServer) RegisterNode(ctx context.Context, req *pb.RegisterNodeRequest) (*pb.RegisterNodeResponse, error) {
	s.network.logger.Info("Node registration request", "nodeID", req.GetNodeId())

	return &pb.RegisterNodeResponse{
		Success: true,
		Message: "Node registered successfully",
	}, nil
}

func (s *NodeServer) GetResources(ctx context.Context, req *pb.GetResourcesRequest) (*pb.GetResourcesResponse, error) {
	return &pb.GetResourcesResponse{
		Resources: &pb.ResourceInfo{
			CpuCores:  4,
			MemoryMb:  8192,
			MaxLayers: 10,
		},
		AvailableLayers: 8,
	}, nil
}

func (s *NodeServer) ProcessInference(ctx context.Context, req *pb.InferenceRequest) (*pb.InferenceResponse, error) {
	return &pb.InferenceResponse{
		Success:       true,
		GeneratedText: "Hello from node " + s.network.nodeID,
	}, nil
}

func (s *NodeServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Healthy:       true,
		Status:        "running",
		UptimeSeconds: 3600,
	}, nil
}
