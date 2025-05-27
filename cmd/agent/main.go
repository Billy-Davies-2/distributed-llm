package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"distributed-llm/internal/agent"
	"distributed-llm/internal/network"
	"distributed-llm/pkg/metrics"
)

func main() {
	var (
		nodeID      = flag.String("node-id", "", "Unique node identifier")
		bindPort    = flag.Int("bind-port", 8080, "Port for gRPC server")
		gossipPort  = flag.Int("gossip-port", 7946, "Port for memberlist gossip")
		metricsPort = flag.Int("metrics-port", 9090, "Port for Prometheus metrics")
		seedNodes   = flag.String("seed-nodes", "", "Comma-separated list of seed nodes (host:port)")
	)
	flag.Parse()

	if *nodeID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			slog.Error("Failed to get hostname", "error", err)
			os.Exit(1)
		}
		*nodeID = hostname
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("Starting distributed LLM agent", "nodeID", *nodeID)

	// Initialize metrics collector
	metricsCollector := metrics.NewMetricsCollector(*nodeID, *metricsPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics collection
	if err := metricsCollector.Start(ctx); err != nil {
		logger.Error("Failed to start metrics collector", "error", err)
		os.Exit(1)
	}
	logger.Info("Metrics server started", "port", *metricsPort)

	// Parse seed nodes
	var seeds []string
	if *seedNodes != "" {
		seeds = strings.Split(*seedNodes, ",")
		for i, seed := range seeds {
			seeds[i] = strings.TrimSpace(seed)
		}
	}

	// Create P2P network with metrics
	network, err := network.NewP2PNetwork(*nodeID, *bindPort, *gossipPort)
	if err != nil {
		logger.Error("Failed to create P2P network", "error", err)
		os.Exit(1)
	}

	// Set metrics collector in network (we'll add this method)
	network.SetMetricsCollector(metricsCollector)

	// Create and configure broadcaster
	broadcaster := agent.NewBroadcaster()
	broadcaster.SetMetricsCollector(metricsCollector)

	// Start broadcaster
	if err := broadcaster.Start(ctx); err != nil {
		logger.Error("Failed to start broadcaster", "error", err)
		os.Exit(1)
	}
	logger.Info("Broadcaster started")

	// Start the network
	if err := network.Start(seeds); err != nil {
		logger.Error("Failed to start P2P network", "error", err)
		os.Exit(1)
	}

	logger.Info("Agent started successfully")
	logger.Info("Node ID", "nodeID", *nodeID)
	logger.Info("gRPC server listening", "port", *bindPort)
	logger.Info("Memberlist gossip", "port", *gossipPort)
	logger.Info("Metrics endpoint", "url", fmt.Sprintf("http://localhost:%d/metrics", *metricsPort))
	if len(seeds) > 0 {
		logger.Info("Seed nodes", "seeds", seeds)
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	logger.Info("Shutting down agent...")

	// Stop metrics collector
	if err := metricsCollector.Stop(); err != nil {
		logger.Error("Error stopping metrics collector", "error", err)
	}

	network.Stop()
	logger.Info("Agent shutdown complete")
}
