package tui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"distributed-llm/pkg/models"
	pb "distributed-llm/proto"
)

type Client struct {
	serverAddr string
	conn       *grpc.ClientConn
	client     pb.NodeServiceClient
}

func NewClient(serverAddr string) *Client {
	return &Client{serverAddr: serverAddr}
}

func (c *Client) Connect() error {
	var err error
	c.conn, err = grpc.NewClient(c.serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.client = pb.NewNodeServiceClient(c.conn)
	return nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) GetResourceInfo() (*models.ResourceInfo, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.GetResources(ctx, &pb.GetResourcesRequest{NodeId: "tui-client"})
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
