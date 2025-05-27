package tui

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"distributed-llm/pkg/models"
)

// AgentDiscovery handles discovery and connection to distributed agents
type AgentDiscovery struct {
	mu           sync.RWMutex
	clients      map[string]*Client
	nodes        map[string]*models.Node
	logger       *slog.Logger
	updateChan   chan []models.Node
	seedNodes    []string
	dockerMode   bool
	k8sNamespace string
}

type DiscoveryConfig struct {
	SeedNodes    []string
	DockerMode   bool
	K8sNamespace string
	UpdateChan   chan []models.Node
}

func NewAgentDiscovery(config DiscoveryConfig) *AgentDiscovery {
	return &AgentDiscovery{
		clients:      make(map[string]*Client),
		nodes:        make(map[string]*models.Node),
		logger:       slog.Default(),
		updateChan:   config.UpdateChan,
		seedNodes:    config.SeedNodes,
		dockerMode:   config.DockerMode,
		k8sNamespace: config.K8sNamespace,
	}
}

// Start begins the discovery process
func (d *AgentDiscovery) Start() error {
	d.logger.Info("Starting agent discovery",
		"dockerMode", d.dockerMode,
		"k8sNamespace", d.k8sNamespace,
		"seedNodes", d.seedNodes)

	// Start discovery based on mode
	if d.dockerMode {
		return d.startDockerDiscovery()
	} else if d.k8sNamespace != "" {
		return d.startK8sDiscovery()
	} else {
		return d.startSeedNodeDiscovery()
	}
}

// startDockerDiscovery discovers agents in Docker containers
func (d *AgentDiscovery) startDockerDiscovery() error {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			d.discoverDockerAgents()
			select {
			case <-ticker.C:
				// Continue loop
			}
		}
	}()

	return nil
}

// startK8sDiscovery discovers agents in Kubernetes cluster
func (d *AgentDiscovery) startK8sDiscovery() error {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			d.discoverK8sAgents()
			select {
			case <-ticker.C:
				// Continue loop
			}
		}
	}()

	return nil
}

// startSeedNodeDiscovery connects to specified seed nodes
func (d *AgentDiscovery) startSeedNodeDiscovery() error {
	for _, seedNode := range d.seedNodes {
		go d.connectToAgent(seedNode)
	}

	// Start periodic discovery of new nodes through gossip
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			d.discoverThroughGossip()
			select {
			case <-ticker.C:
				// Continue loop
			}
		}
	}()

	return nil
}

// discoverDockerAgents finds agents running in Docker containers
func (d *AgentDiscovery) discoverDockerAgents() {
	// Common Docker service names and ports
	commonNames := []string{
		"distributed-llm-agent",
		"agent",
		"llm-agent",
	}

	ports := []int{8080, 8081, 8082, 8083}

	for _, name := range commonNames {
		for _, port := range ports {
			address := fmt.Sprintf("%s:%d", name, port)
			if d.tryConnectToAgent(address) {
				d.logger.Info("Found Docker agent", "address", address)
			}
		}
	}

	// Also try localhost ports for docker-compose
	for _, port := range ports {
		address := fmt.Sprintf("localhost:%d", port)
		if d.tryConnectToAgent(address) {
			d.logger.Info("Found localhost agent", "address", address)
		}
	}
}

// discoverK8sAgents finds agents through Kubernetes service discovery
func (d *AgentDiscovery) discoverK8sAgents() {
	// Try common Kubernetes service patterns
	services := []string{
		fmt.Sprintf("distributed-llm-agent.%s.svc.cluster.local:8080", d.k8sNamespace),
		fmt.Sprintf("agent.%s.svc.cluster.local:8080", d.k8sNamespace),
		fmt.Sprintf("llm-agent.%s.svc.cluster.local:8080", d.k8sNamespace),
	}

	for _, service := range services {
		if d.tryConnectToAgent(service) {
			d.logger.Info("Found Kubernetes agent", "service", service)
		}
	}

	// Try to discover through DNS SRV records
	d.discoverThroughSRV()
}

// discoverThroughSRV uses DNS SRV records for service discovery
func (d *AgentDiscovery) discoverThroughSRV() {
	_, srvs, err := net.LookupSRV("grpc", "tcp", fmt.Sprintf("distributed-llm.%s.svc.cluster.local", d.k8sNamespace))
	if err != nil {
		d.logger.Debug("No SRV records found", "error", err)
		return
	}

	for _, srv := range srvs {
		address := fmt.Sprintf("%s:%d", strings.TrimSuffix(srv.Target, "."), srv.Port)
		if d.tryConnectToAgent(address) {
			d.logger.Info("Found agent via SRV", "address", address)
		}
	}
}

// discoverThroughGossip discovers new nodes through existing connections
func (d *AgentDiscovery) discoverThroughGossip() {
	d.mu.RLock()
	clients := make([]*Client, 0, len(d.clients))
	for _, client := range d.clients {
		clients = append(clients, client)
	}
	d.mu.RUnlock()

	for _, client := range clients {
		peers, err := d.getPeersFromAgent(client)
		if err != nil {
			d.logger.Debug("Failed to get peers", "error", err)
			continue
		}

		for _, peer := range peers {
			if d.tryConnectToAgent(peer) {
				d.logger.Info("Found agent via gossip", "address", peer)
			}
		}
	}
}

// tryConnectToAgent attempts to connect to an agent and returns true if successful
func (d *AgentDiscovery) tryConnectToAgent(address string) bool {
	d.mu.RLock()
	_, exists := d.clients[address]
	d.mu.RUnlock()

	if exists {
		return true // Already connected
	}

	client := NewClient(address)
	if err := client.Connect(); err != nil {
		return false
	}

	// Try to get resources to verify connection
	resources, err := client.GetResourceInfo()
	if err != nil {
		client.Close()
		return false
	}

	// Create node from resources
	node := &models.Node{
		ID:        fmt.Sprintf("agent-%s", address),
		Address:   strings.Split(address, ":")[0],
		Port:      8080, // Default port
		Status:    models.NodeStatusOnline,
		Resources: *resources,
		LastSeen:  time.Now(),
	}

	d.mu.Lock()
	d.clients[address] = client
	d.nodes[address] = node
	d.mu.Unlock()

	// Start resource monitoring for this agent
	go d.monitorAgent(address, client)

	// Notify UI of updated nodes
	d.notifyNodesUpdate()

	return true
}

// connectToAgent establishes connection to a specific agent
func (d *AgentDiscovery) connectToAgent(address string) {
	d.logger.Info("Connecting to agent", "address", address)

	for {
		if d.tryConnectToAgent(address) {
			break
		}

		d.logger.Debug("Failed to connect to agent, retrying", "address", address)
		time.Sleep(5 * time.Second)
	}
}

// monitorAgent continuously monitors an agent's health and resources
func (d *AgentDiscovery) monitorAgent(address string, client *Client) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			resources, err := client.GetResourceInfo()
			if err != nil {
				d.logger.Warn("Agent became unreachable", "address", address, "error", err)
				d.removeAgent(address)
				return
			}

			d.mu.Lock()
			if node, exists := d.nodes[address]; exists {
				node.Resources = *resources
				node.LastSeen = time.Now()
				node.Status = models.NodeStatusOnline
			}
			d.mu.Unlock()

			d.notifyNodesUpdate()
		}
	}
}

// removeAgent removes a disconnected agent
func (d *AgentDiscovery) removeAgent(address string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if client, exists := d.clients[address]; exists {
		client.Close()
		delete(d.clients, address)
	}

	delete(d.nodes, address)
	d.notifyNodesUpdate()
}

// notifyNodesUpdate sends updated node list to the UI
func (d *AgentDiscovery) notifyNodesUpdate() {
	if d.updateChan == nil {
		return
	}

	d.mu.RLock()
	nodes := make([]models.Node, 0, len(d.nodes))
	for _, node := range d.nodes {
		nodes = append(nodes, *node)
	}
	d.mu.RUnlock()

	select {
	case d.updateChan <- nodes:
	default:
		// Channel full, skip update
	}
}

// getPeersFromAgent gets peer list from an agent using protobuf
func (d *AgentDiscovery) getPeersFromAgent(client *Client) ([]string, error) {
	return client.GetPeers()
}

// GetNodes returns current list of discovered nodes
func (d *AgentDiscovery) GetNodes() []models.Node {
	d.mu.RLock()
	defer d.mu.RUnlock()

	nodes := make([]models.Node, 0, len(d.nodes))
	for _, node := range d.nodes {
		nodes = append(nodes, *node)
	}

	return nodes
}

// GetClient returns a client for the specified address
func (d *AgentDiscovery) GetClient(address string) (*Client, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	client, exists := d.clients[address]
	return client, exists
}

// Stop shuts down the discovery system
func (d *AgentDiscovery) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, client := range d.clients {
		client.Close()
	}

	d.clients = make(map[string]*Client)
	d.nodes = make(map[string]*models.Node)
}
