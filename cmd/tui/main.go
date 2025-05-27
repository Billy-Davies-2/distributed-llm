package main

import (
	"log/slog"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"distributed-llm/internal/tui"
	"distributed-llm/pkg/models"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("Starting Distributed LLM TUI client...")

	// Create the Bubble Tea model
	model := tui.NewModel()

	// Mock some data for demonstration
	go func() {
		time.Sleep(2 * time.Second)

		// Mock nodes data
		mockNodes := []models.Node{
			{
				ID:      "node-1",
				Address: "192.168.1.100",
				Port:    8080,
				Status:  models.NodeStatusOnline,
				Resources: models.ResourceInfo{
					CPUCores:   8,
					MemoryMB:   16384,
					MaxLayers:  20,
					UsedLayers: 5,
					GPUs: []models.GPUInfo{
						{Name: "NVIDIA RTX 4090", MemoryMB: 24576, UUID: "gpu-uuid-1"},
					},
				},
				LastSeen: time.Now(),
			},
			{
				ID:      "node-2",
				Address: "192.168.1.101",
				Port:    8080,
				Status:  models.NodeStatusBusy,
				Resources: models.ResourceInfo{
					CPUCores:   4,
					MemoryMB:   8192,
					MaxLayers:  10,
					UsedLayers: 8,
				},
				LastSeen: time.Now(),
			},
		}

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
		}

		model.UpdateNodes(mockNodes)
		model.UpdateModels(mockModels)
	}()

	// Start the TUI
	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		logger.Error("Error running TUI", "error", err)
		os.Exit(1)
	}

	logger.Info("TUI client shutting down...")
}
