package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	pb "distributed-llm/proto"
)

// Simple mock service
type SimpleMockNodeService struct {
	pb.UnimplementedNodeServiceServer
}

func (m *SimpleMockNodeService) GetResources(ctx context.Context, req *pb.GetResourcesRequest) (*pb.GetResourcesResponse, error) {
	fmt.Println("Mock service GetResources called!")
	return &pb.GetResourcesResponse{
		Resources: &pb.ResourceInfo{
			CpuCores:   8,
			MemoryMb:   16384,
			MaxLayers:  20,
			UsedLayers: 5,
		},
		AvailableLayers: 15,
	}, nil
}

func TestSimpleMockConnection(t *testing.T) {
	// Create mock server
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	mockService := &SimpleMockNodeService{}
	pb.RegisterNodeServiceServer(server, mockService)

	go func() {
		if err := server.Serve(listener); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Create client
	conn, err := grpc.NewClient("test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer conn.Close()
	defer server.Stop()
	defer listener.Close()

	// Test the call
	client := pb.NewNodeServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetResources(ctx, &pb.GetResourcesRequest{NodeId: "test"})
	if err != nil {
		t.Fatalf("GetResources failed: %v", err)
	}

	if resp.Resources.CpuCores != 8 {
		t.Errorf("Expected 8 CPU cores, got %d", resp.Resources.CpuCores)
	}

	fmt.Println("Test passed!")
}
