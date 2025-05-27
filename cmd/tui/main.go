package main

import (
	"flag"
	"log/slog"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"distributed-llm/internal/tui"
	"distributed-llm/pkg/models"
)

func main() {
	var (
		seedNodes    = flag.String("seed-nodes", "", "Comma-separated list of seed nodes (host:port)")
		dockerMode   = flag.Bool("docker", false, "Use Docker service discovery")
		k8sNamespace = flag.String("k8s-namespace", "default", "Kubernetes namespace for service discovery")
		logLevel     = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	// Configure logging
	var level slog.Level
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	logger.Info("Starting Distributed LLM TUI client...",
		"dockerMode", *dockerMode,
		"k8sNamespace", *k8sNamespace,
		"seedNodes", *seedNodes)

	// Create channels for agent communication
	nodeUpdateChan := make(chan []models.Node, 10)
	modelUpdateChan := make(chan []models.Model, 10)

	// Create the Bubble Tea model
	model := tui.NewModelWithChannels(nodeUpdateChan, modelUpdateChan)

	// Parse seed nodes
	var seedNodesList []string
	if *seedNodes != "" {
		seedNodesList = strings.Split(*seedNodes, ",")
		for i, node := range seedNodesList {
			seedNodesList[i] = strings.TrimSpace(node)
		}
	}

	// Set up default seed nodes if none provided
	if len(seedNodesList) == 0 {
		if *dockerMode {
			// Default Docker service names
			seedNodesList = []string{
				"distributed-llm-agent:8080",
				"agent:8080",
				"localhost:8080",
				"localhost:8081",
				"localhost:8082",
			}
		} else if *k8sNamespace != "" {
			// Default Kubernetes service names
			seedNodesList = []string{
				"distributed-llm-agent." + *k8sNamespace + ".svc.cluster.local:8080",
				"agent." + *k8sNamespace + ".svc.cluster.local:8080",
			}
		} else {
			// Default local development
			seedNodesList = []string{
				"localhost:8080",
				"127.0.0.1:8080",
			}
		}
	}

	// Create and start agent discovery
	discovery := tui.NewAgentDiscovery(tui.DiscoveryConfig{
		SeedNodes:    seedNodesList,
		DockerMode:   *dockerMode,
		K8sNamespace: *k8sNamespace,
		UpdateChan:   nodeUpdateChan,
	})

	if err := discovery.Start(); err != nil {
		logger.Error("Failed to start agent discovery", "error", err)
		os.Exit(1)
	}

	// Start mock models for demonstration (until we have model discovery)
	go func() {
		time.Sleep(3 * time.Second)

		mockModels := []models.Model{
			{
				ID:         "llama-7b",
				Name:       "Llama 2 7B",
				Version:    "1.0",
				LayerCount: 32,
				FilePath:   "/models/llama-2-7b.gguf",
				Size:       7000000000,
			},
			{
				ID:         "mistral-7b",
				Name:       "Mistral 7B",
				Version:    "0.1",
				LayerCount: 32,
				FilePath:   "/models/mistral-7b.gguf",
				Size:       7200000000,
			},
			{
				ID:         "gpt-3.5-turbo",
				Name:       "GPT-3.5 Turbo",
				Version:    "1.0",
				LayerCount: 24,
				FilePath:   "/models/gpt-3.5-turbo.gguf",
				Size:       6500000000,
			},
		}

		select {
		case modelUpdateChan <- mockModels:
		default:
		}
	}()

	// Set up graceful shutdown
	go func() {
		// This would be replaced with proper signal handling
		// For now, let discovery run indefinitely
		select {}
	}()

	// Start the TUI
	program := tea.NewProgram(model, tea.WithAltScreen())

	logger.Info("Starting TUI interface...")
	if _, err := program.Run(); err != nil {
		logger.Error("Error running TUI", "error", err)
		discovery.Stop()
		os.Exit(1)
	}

	logger.Info("TUI client shutting down...")
	discovery.Stop()
}
