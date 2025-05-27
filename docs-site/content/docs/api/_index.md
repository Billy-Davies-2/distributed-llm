---
title: "API Reference"
linkTitle: "API Reference"
weight: 40
description: >
  Complete gRPC API documentation for all services
---

## Overview

Distributed LLM provides three main gRPC services for cluster management and communication:

- **NodeService**: Core node operations and health management
- **DiscoveryService**: Cluster discovery and membership
- **TUIService**: Terminal interface backend services

All services support gzip compression and use Protocol Buffers for message serialization.

## NodeService

Core service for node management and health reporting.

### GetNodeInfo

Retrieves information about a specific node.

```protobuf
rpc GetNodeInfo(NodeInfoRequest) returns (NodeInfoResponse);
```

**Request:**
```protobuf
message NodeInfoRequest {
    string node_id = 1;
}
```

**Response:**
```protobuf
message NodeInfoResponse {
    NodeInfo node = 1;
    bool success = 2;
    string message = 3;
}
```

**Example:**
```bash
grpcurl -plaintext localhost:8080 \
  distributed_llm.NodeService/GetNodeInfo \
  -d '{"node_id": "agent-1"}'
```

### GetPeers

Lists all known peer nodes in the cluster.

```protobuf
rpc GetPeers(PeersRequest) returns (PeersResponse);
```

**Request:**
```protobuf
message PeersRequest {
    string requester_id = 1;
    bool include_self = 2;
}
```

**Response:**
```protobuf
message PeersResponse {
    repeated NodeInfo peers = 1;
    int32 total_count = 2;
}
```

### GetMetrics

Retrieves current node metrics.

```protobuf
rpc GetMetrics(MetricsRequest) returns (MetricsResponse);
```

**Request:**
```protobuf
message MetricsRequest {
    string node_id = 1;
    repeated string metric_names = 2;
}
```

**Response:**
```protobuf
message MetricsResponse {
    repeated MetricValue metrics = 1;
    int64 timestamp = 2;
    bool success = 3;
}
```

### StreamMetrics

Provides real-time streaming of node metrics.

```protobuf
rpc StreamMetrics(MetricsStreamRequest) returns (stream MetricsStreamResponse);
```

**Request:**
```protobuf
message MetricsStreamRequest {
    string node_id = 1;
    int32 interval_seconds = 2;
    repeated string metric_names = 3;
}
```

**Response Stream:**
```protobuf
message MetricsStreamResponse {
    repeated MetricValue metrics = 1;
    int64 timestamp = 2;
}
```

## DiscoveryService

Handles cluster formation, node discovery, and membership management.

### DiscoverNodes

Discovers new nodes in the cluster.

```protobuf
rpc DiscoverNodes(DiscoveryRequest) returns (DiscoveryResponse);
```

**Request:**
```protobuf
message DiscoveryRequest {
    string requester_id = 1;
    repeated string known_nodes = 2;
    ResourceFilter filter = 3;
}
```

**Response:**
```protobuf
message DiscoveryResponse {
    repeated NodeInfo discovered_nodes = 1;
    bool success = 2;
    string message = 3;
}
```

**Example:**
```bash
grpcurl -plaintext localhost:8080 \
  distributed_llm.DiscoveryService/DiscoverNodes \
  -d '{"requester_id": "tui-client-1", "known_nodes": []}'
```

### RegisterWithCluster

Registers a new node with the cluster.

```protobuf
rpc RegisterWithCluster(ClusterJoinRequest) returns (ClusterJoinResponse);
```

**Request:**
```protobuf
message ClusterJoinRequest {
    string node_id = 1;
    string address = 2;
    int32 port = 3;
    ResourceInfo resources = 4;
    repeated string seed_nodes = 5;
}
```

**Response:**
```protobuf
message ClusterJoinResponse {
    bool success = 1;
    string message = 2;
    repeated NodeInfo existing_nodes = 3;
    string cluster_id = 4;
}
```

### LeaveCluster

Gracefully removes a node from the cluster.

```protobuf
rpc LeaveCluster(ClusterLeaveRequest) returns (ClusterLeaveResponse);
```

**Request:**
```protobuf
message ClusterLeaveRequest {
    string node_id = 1;
    string reason = 2;
}
```

**Response:**
```protobuf
message ClusterLeaveResponse {
    bool success = 1;
    string message = 2;
}
```

### GetClusterInfo

Retrieves comprehensive cluster information.

```protobuf
rpc GetClusterInfo(ClusterInfoRequest) returns (ClusterInfoResponse);
```

**Response:**
```protobuf
message ClusterInfoResponse {
    string cluster_id = 1;
    repeated NodeInfo nodes = 2;
    repeated ModelInfo models = 3;
    ClusterMetrics metrics = 4;
}
```

## TUIService

Backend service for the Terminal User Interface.

### GetNodeList

Retrieves formatted node list for TUI display.

```protobuf
rpc GetNodeList(NodeListRequest) returns (NodeListResponse);
```

**Request:**
```protobuf
message NodeListRequest {
    string requester_id = 1;
    bool include_metrics = 2;
    NodeFilter filter = 3;
}
```

**Response:**
```protobuf
message NodeListResponse {
    repeated NodeInfo nodes = 1;
    ClusterMetrics cluster_metrics = 2;
}
```

### GetModelList

Retrieves available models in the cluster.

```protobuf
rpc GetModelList(ModelListRequest) returns (ModelListResponse);
```

**Response:**
```protobuf
message ModelListResponse {
    repeated ModelInfo models = 1;
}
```

### StreamUpdates

Provides real-time updates for the TUI interface.

```protobuf
rpc StreamUpdates(UpdateStreamRequest) returns (stream ClusterUpdate);
```

**Request:**
```protobuf
message UpdateStreamRequest {
    string requester_id = 1;
    int32 interval_seconds = 2;
    repeated string update_types = 3; // "nodes", "models", "metrics"
}
```

**Response Stream:**
```protobuf
message ClusterUpdate {
    string update_type = 1;
    int64 timestamp = 2;
    repeated NodeInfo nodes = 3;
    repeated ModelInfo models = 4;
    ClusterMetrics metrics = 5;
}
```

### ExecuteCommand

Executes administrative commands on the cluster.

```protobuf
rpc ExecuteCommand(CommandRequest) returns (CommandResponse);
```

**Request:**
```protobuf
message CommandRequest {
    string requester_id = 1;
    string command = 2;
    repeated string args = 3;
    map<string, string> options = 4;
}
```

**Response:**
```protobuf
message CommandResponse {
    bool success = 1;
    string output = 2;
    string error = 3;
    int32 exit_code = 4;
}
```

## Data Types

### NodeInfo

```protobuf
message NodeInfo {
    string node_id = 1;
    string address = 2;
    int32 port = 3;
    ResourceInfo resources = 4;
    string status = 5;
    int64 last_seen = 6;
}
```

### ResourceInfo

```protobuf
message ResourceInfo {
    int64 cpu_cores = 1;
    int64 memory_mb = 2;
    repeated GPUInfo gpus = 3;
    int32 max_layers = 4;
    int32 used_layers = 5;
}
```

### GPUInfo

```protobuf
message GPUInfo {
    string name = 1;
    int64 memory_mb = 2;
    string uuid = 3;
}
```

### ModelInfo

```protobuf
message ModelInfo {
    string id = 1;
    string name = 2;
    string version = 3;
    int32 layer_count = 4;
    string file_path = 5;
    int64 size_bytes = 6;
    repeated string node_assignments = 7;
}
```

### ClusterMetrics

```protobuf
message ClusterMetrics {
    int32 total_nodes = 1;
    int32 healthy_nodes = 2;
    int64 total_memory_mb = 3;
    int64 available_memory_mb = 4;
    int32 total_gpus = 5;
    int32 total_layers = 6;
    int32 allocated_layers = 7;
    float cluster_utilization = 8;
}
```

## gRPC Client Examples

### Go Client

```go
package main

import (
    "context"
    "log"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/encoding/gzip"
    
    pb "distributed-llm/proto"
)

func main() {
    // Connect with compression
    conn, err := grpc.Dial("localhost:8080",
        grpc.WithInsecure(),
        grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    client := pb.NewNodeServiceClient(conn)
    
    // Get node info
    resp, err := client.GetNodeInfo(context.Background(), &pb.NodeInfoRequest{
        NodeId: "agent-1",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Node: %s, Status: %s", resp.Node.NodeId, resp.Node.Status)
}
```

### Python Client

```python
import grpc
import node_pb2
import node_pb2_grpc

def main():
    # Create channel with compression
    channel = grpc.insecure_channel('localhost:8080',
        options=[('grpc.default_compression_algorithm', grpc.Compression.Gzip)]
    )
    
    stub = node_pb2_grpc.NodeServiceStub(channel)
    
    # Get node info
    request = node_pb2.NodeInfoRequest(node_id='agent-1')
    response = stub.GetNodeInfo(request)
    
    print(f"Node: {response.node.node_id}, Status: {response.node.status}")

if __name__ == '__main__':
    main()
```

## Error Codes

Common gRPC status codes used:

- `OK (0)`: Success
- `INVALID_ARGUMENT (3)`: Invalid request parameters
- `NOT_FOUND (5)`: Node or resource not found
- `UNAVAILABLE (14)`: Service temporarily unavailable
- `INTERNAL (13)`: Internal server error

## Authentication

Currently, the services operate without authentication for development. Production deployments should implement:

- mTLS for inter-service communication
- JWT tokens for client authentication
- RBAC for authorization
