package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"distributed-llm/pkg/models"

	tea "github.com/charmbracelet/bubbletea"
)

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

func TestNewModel(t *testing.T) {
	model := NewModel()

	if model.currentTab != TabNodes {
		t.Errorf("Expected initial tab to be TabNodes, got %v", model.currentTab)
	}

	if len(model.nodes) != 0 {
		t.Errorf("Expected empty nodes slice, got %d nodes", len(model.nodes))
	}

	if len(model.modelList) != 0 {
		t.Errorf("Expected empty models slice, got %d models", len(model.modelList))
	}

	if model.selectedNode != 0 {
		t.Errorf("Expected selectedNode to be 0, got %d", model.selectedNode)
	}
}

func TestModelInit(t *testing.T) {
	model := NewModel()
	cmd := model.Init()

	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestModelUpdate(t *testing.T) {
	tests := []struct {
		name       string
		msg        tea.Msg
		expectTab  Tab
		expectQuit bool
	}{
		{
			name:       "Tab key switches tabs",
			msg:        tea.KeyMsg{Type: tea.KeyTab},
			expectTab:  TabModels,
			expectQuit: false,
		},
		{
			name:       "Q key quits",
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			expectTab:  TabNodes,
			expectQuit: true,
		},
		{
			name:       "Ctrl+C quits",
			msg:        tea.KeyMsg{Type: tea.KeyCtrlC},
			expectTab:  TabNodes,
			expectQuit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := NewModel()
			updatedModel, cmd := testModel.Update(tt.msg)

			m := updatedModel.(Model)
			if m.currentTab != tt.expectTab {
				t.Errorf("Expected tab %v, got %v", tt.expectTab, m.currentTab)
			}

			if tt.expectQuit {
				if cmd == nil {
					t.Error("Expected quit command but got nil")
				}
				// Note: We can't easily test if cmd is tea.Quit without reflection
			}
		})
	}
}

func TestModelUpdateNavigation(t *testing.T) {
	model := NewModel()

	// Add some test nodes
	nodes := []models.Node{
		{ID: "node1", Status: models.NodeStatusOnline},
		{ID: "node2", Status: models.NodeStatusOnline},
		{ID: "node3", Status: models.NodeStatusOffline},
	}
	model.UpdateNodes(nodes)

	// Test down navigation
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	m := updatedModel.(Model)
	if m.selectedNode != 1 {
		t.Errorf("Expected selectedNode 1, got %d", m.selectedNode)
	}

	// Test up navigation
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updatedModel.(Model)
	if m.selectedNode != 0 {
		t.Errorf("Expected selectedNode 0, got %d", m.selectedNode)
	}

	// Test navigation bounds (up from 0)
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updatedModel.(Model)
	if m.selectedNode != 0 {
		t.Errorf("Expected selectedNode to stay at 0, got %d", m.selectedNode)
	}

	// Navigate to last node
	m.selectedNode = 2
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updatedModel.(Model)
	if m.selectedNode != 2 {
		t.Errorf("Expected selectedNode to stay at 2, got %d", m.selectedNode)
	}
}

func TestModelUpdateNodes(t *testing.T) {
	model := NewModel()

	nodes := []models.Node{
		{
			ID:      "node1",
			Address: "192.168.1.100",
			Port:    8080,
			Status:  models.NodeStatusOnline,
			Resources: models.ResourceInfo{
				CPUCores: 4,
				MemoryMB: 8192,
			},
		},
		{
			ID:      "node2",
			Address: "192.168.1.101",
			Port:    8080,
			Status:  models.NodeStatusBusy,
			Resources: models.ResourceInfo{
				CPUCores: 8,
				MemoryMB: 16384,
			},
		},
	}

	model.UpdateNodes(nodes)

	if len(model.nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(model.nodes))
	}

	if model.nodes[0].ID != "node1" {
		t.Errorf("Expected first node ID 'node1', got '%s'", model.nodes[0].ID)
	}

	if model.nodes[1].Status != models.NodeStatusBusy {
		t.Errorf("Expected second node status 'busy', got '%s'", model.nodes[1].Status)
	}
}

func TestModelUpdateModels(t *testing.T) {
	model := NewModel()

	models := []models.Model{
		{
			ID:         "model1",
			Name:       "GPT-4",
			Version:    "1.0",
			Size:       7 * 1024 * 1024 * 1024, // 7GB
			LayerCount: 96,
			FilePath:   "/models/gpt4.bin",
		},
		{
			ID:         "model2",
			Name:       "LLaMA-2",
			Version:    "2.0",
			Size:       13 * 1024 * 1024 * 1024, // 13GB
			LayerCount: 80,
			FilePath:   "/models/llama2.bin",
		},
	}

	model.UpdateModels(models)

	if len(model.modelList) != 2 {
		t.Errorf("Expected 2 models, got %d", len(model.modelList))
	}

	if model.modelList[0].Name != "GPT-4" {
		t.Errorf("Expected first model name 'GPT-4', got '%s'", model.modelList[0].Name)
	}

	if model.modelList[1].LayerCount != 80 {
		t.Errorf("Expected second model layer count 80, got %d", model.modelList[1].LayerCount)
	}
}

func TestModelView(t *testing.T) {
	model := NewModel()
	model.width = 80
	model.height = 24

	// Test view with no data
	view := model.View()
	if view == "" {
		t.Error("View should not be empty")
	}

	// Should contain retro elements
	if !strings.Contains(view, "DOS-LLVM") {
		t.Error("View should contain DOS-LLVM branding")
	}

	if !strings.Contains(view, "[NODES]") {
		t.Error("View should contain nodes tab")
	}

	// Test view with zero dimensions
	model.width = 0
	view = model.View()
	if !strings.Contains(view, "Loading") {
		t.Error("Should show loading message when width is 0")
	}
}

func TestTabSwitching(t *testing.T) {
	model := NewModel()

	// Start at TabNodes
	if model.currentTab != TabNodes {
		t.Errorf("Expected initial tab TabNodes, got %v", model.currentTab)
	}

	// Switch to TabModels
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	m := updatedModel.(Model)
	if m.currentTab != TabModels {
		t.Errorf("Expected TabModels, got %v", m.currentTab)
	}

	// Switch to TabInference
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedModel.(Model)
	if m.currentTab != TabInference {
		t.Errorf("Expected TabInference, got %v", m.currentTab)
	}

	// Switch back to TabNodes (wrap around)
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedModel.(Model)
	if m.currentTab != TabNodes {
		t.Errorf("Expected TabNodes (wrap around), got %v", m.currentTab)
	}
}

func TestRenderMethods(t *testing.T) {
	model := NewModel()
	model.width = 80
	model.height = 24

	// Test nodes tab rendering
	model.currentTab = TabNodes
	nodesView := model.renderNodesTab()
	cleanNodesView := stripANSI(nodesView)
	if !strings.Contains(cleanNodesView, "CLUSTER NODES STATUS") {
		t.Error("Nodes view should contain cluster nodes header")
	}

	// Test models tab rendering
	model.currentTab = TabModels
	modelsView := model.renderModelsTab()
	cleanModelsView := stripANSI(modelsView)
	if !strings.Contains(cleanModelsView, "MODEL REPOSITORY") {
		t.Error("Models view should contain model repository header")
	}
	// Test inference tab rendering
	model.currentTab = TabInference

	// Disable glitch effects for reliable testing
	model.glitch.SetEnabled(false)

	inferenceView := model.renderInferenceTab()
	cleanInferenceView := stripANSI(inferenceView)

	if !strings.Contains(cleanInferenceView, "INFERENCE CONSOLE") {
		t.Error("Inference view should contain inference console header")
	}
}

func TestTickMsg(t *testing.T) {
	model := NewModel()
	now := time.Now()

	updatedModel, cmd := model.Update(TickMsg(now))
	m := updatedModel.(Model)

	if m.lastUpdate != now {
		t.Error("TickMsg should update lastUpdate time")
	}

	if cmd == nil {
		t.Error("TickMsg should return a command")
	}
}

func TestWindowSizeMsg(t *testing.T) {
	model := NewModel()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	if m.width != 120 {
		t.Errorf("Expected width 120, got %d", m.width)
	}

	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}

// Benchmark tests
func BenchmarkModelUpdate(b *testing.B) {
	model := NewModel()
	msg := tea.KeyMsg{Type: tea.KeyTab}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Update(msg)
	}
}

func BenchmarkModelView(b *testing.B) {
	model := NewModel()
	model.width = 80
	model.height = 24

	// Add some test data
	nodes := []models.Node{
		{ID: "node1", Status: models.NodeStatusOnline},
		{ID: "node2", Status: models.NodeStatusBusy},
	}
	model.UpdateNodes(nodes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.View()
	}
}

func BenchmarkRenderNodes(b *testing.B) {
	model := NewModel()
	model.width = 80

	// Add test nodes
	nodes := make([]models.Node, 10)
	for i := 0; i < 10; i++ {
		nodes[i] = models.Node{
			ID:     fmt.Sprintf("node%d", i),
			Status: models.NodeStatusOnline,
			Resources: models.ResourceInfo{
				CPUCores: 8,
				MemoryMB: 16384,
			},
		}
	}
	model.UpdateNodes(nodes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.renderNodesTab()
	}
}
