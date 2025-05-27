package agent

import (
	"context"
	"distributed-llm/pkg/models"
	"sync"
	"time"
)

// MetricsCollector interface for dependency injection
type MetricsCollector interface {
	UpdateNodeResources(resources models.ResourceInfo)
	UpdateNodeStatus(status models.NodeStatus)
	UpdateNetworkConnections(count int)
	RecordNetworkMessage(direction, messageType string)
}

// Broadcaster handles resource broadcasting and node management
type Broadcaster struct {
	mu               sync.RWMutex
	resources        models.ResourceInfo
	nodes            []models.Node
	listeners        []chan models.ResourceInfo
	metricsCollector MetricsCollector
}

// NewBroadcaster creates a new broadcaster instance
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		resources: GetResourceInfo(),
		nodes:     []models.Node{},
		listeners: []chan models.ResourceInfo{},
	}
}

// SetMetricsCollector sets the metrics collector for the broadcaster
func (b *Broadcaster) SetMetricsCollector(collector MetricsCollector) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.metricsCollector = collector
}

// Start begins the broadcasting service
func (b *Broadcaster) Start(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.broadcast()
			}
		}
	}()
	return nil
}

// UpdateResources updates the current resource information
func (b *Broadcaster) UpdateResources(resources models.ResourceInfo) {
	b.mu.Lock()
	b.resources = resources
	// Create a copy of listeners while holding the lock
	listeners := make([]chan models.ResourceInfo, len(b.listeners))
	copy(listeners, b.listeners)
	resourcesCopy := b.resources
	collector := b.metricsCollector
	b.mu.Unlock()

	// Update metrics if collector is available
	if collector != nil {
		collector.UpdateNodeResources(resources)
	}

	// Send updates without holding the lock
	for _, listener := range listeners {
		select {
		case listener <- resourcesCopy:
		default:
			// Don't block if channel is full
		}
	}
}

// GetResources returns the current resource information
func (b *Broadcaster) GetResources() models.ResourceInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.resources
}

// Subscribe adds a listener for resource updates
func (b *Broadcaster) Subscribe(ch chan models.ResourceInfo) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.listeners = append(b.listeners, ch)
}

// AddNode adds a node to the cluster
func (b *Broadcaster) AddNode(node models.Node) {
	b.mu.Lock()
	b.nodes = append(b.nodes, node)
	nodeCount := len(b.nodes)
	collector := b.metricsCollector
	b.mu.Unlock()

	// Update metrics if collector is available
	if collector != nil {
		collector.UpdateNetworkConnections(nodeCount)
		collector.RecordNetworkMessage("incoming", "node_added")
	}
}

// GetNodes returns all known nodes
func (b *Broadcaster) GetNodes() []models.Node {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return append([]models.Node{}, b.nodes...)
}

// broadcast sends resource updates to all listeners
func (b *Broadcaster) broadcast() {
	b.mu.RLock()
	// Create copies while holding the lock
	listeners := make([]chan models.ResourceInfo, len(b.listeners))
	copy(listeners, b.listeners)
	resourcesCopy := b.resources
	collector := b.metricsCollector
	b.mu.RUnlock()

	// Record broadcast metric
	if collector != nil {
		collector.RecordNetworkMessage("outgoing", "resource_broadcast")
	}

	// Send updates without holding the lock
	for _, listener := range listeners {
		select {
		case listener <- resourcesCopy:
		default:
			// Don't block if channel is full
		}
	}
}
