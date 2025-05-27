package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"distributed-llm/pkg/models"
)

// MS-DOS inspired color palette with cool-retro-term green theme
var (
	// Primary green colors inspired by classic terminals
	brightGreen  = lipgloss.Color("#00FF00") // Classic bright green
	darkGreen    = lipgloss.Color("#00AA00") // Darker green
	paleGreen    = lipgloss.Color("#00DD88") // Pale green
	black        = lipgloss.Color("#000000") // Pure black background
	dimGreen     = lipgloss.Color("#008800") // Dimmed green
	errorRed     = lipgloss.Color("#FF4444") // Error red (classic CRT red)
	warningAmber = lipgloss.Color("#FFAA00") // Warning amber

	// Title bar with classic green-on-black terminal style
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(black).
			Background(brightGreen).
			Padding(0, 2).
			MarginBottom(1)

	// Headers with bright green, bold text
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(brightGreen).
			Underline(true)

	// Node containers with classic terminal box drawing
	nodeStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingRight(2).
			PaddingTop(1).
			PaddingBottom(1).
			Border(lipgloss.ThickBorder()).
			BorderForeground(darkGreen).
			Foreground(paleGreen)

	// Status indicators with appropriate colors
	statusOnlineStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(brightGreen)

	statusOfflineStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(errorRed)

	statusBusyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(warningAmber)

	// Selected node highlight
	selectedNodeStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				PaddingRight(2).
				PaddingTop(1).
				PaddingBottom(1).
				Border(lipgloss.DoubleBorder()).
				BorderForeground(brightGreen).
				Foreground(brightGreen).
				Bold(true)

	// Tab styles for retro look
	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Background(brightGreen).
			Foreground(black).
			Bold(true)

	inactiveTabStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Background(dimGreen).
				Foreground(black)

	// Footer style
	footerStyle = lipgloss.NewStyle().
			Foreground(darkGreen).
			MarginTop(1)

	// ASCII art style
	asciiStyle = lipgloss.NewStyle().
			Foreground(darkGreen).
			Bold(true)
)

type Tab int

const (
	TabNodes Tab = iota
	TabModels
	TabInference
)

type Model struct {
	currentTab   Tab
	nodes        []models.Node
	modelList    []models.Model
	selectedNode int
	width        int
	height       int
	lastUpdate   time.Time
}

type TickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			m.currentTab = (m.currentTab + 1) % 3

		case "up", "k":
			if m.selectedNode > 0 {
				m.selectedNode--
			}

		case "down", "j":
			if m.selectedNode < len(m.nodes)-1 {
				m.selectedNode++
			}
		}

	case TickMsg:
		m.lastUpdate = time.Time(msg)
		return m, tickCmd()
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return asciiStyle.Render("Loading DOS-LLVM v1.0...")
	}

	var content strings.Builder

	// ASCII art title banner
	banner := `
██████╗  ██████╗ ███████╗      ██╗     ██╗     ██╗   ██╗███╗   ███╗
██╔══██╗██╔═══██╗██╔════╝      ██║     ██║     ██║   ██║████╗ ████║
██║  ██║██║   ██║███████╗█████╗██║     ██║     ██║   ██║██╔████╔██║
██║  ██║██║   ██║╚════██║╚════╝██║     ██║     ╚██╗ ██╔╝██║╚██╔╝██║
██████╔╝╚██████╔╝███████║      ███████╗███████╗ ╚████╔╝ ██║ ╚═╝ ██║
╚═════╝  ╚═════╝ ╚══════╝      ╚══════╝╚══════╝  ╚═══╝  ╚═╝     ╚═╝
                    Distributed Large Language Model Cluster v1.0`

	title := titleStyle.Render("DOS-LLVM CLUSTER MANAGER")
	content.WriteString(asciiStyle.Render(banner))
	content.WriteString("\n")
	content.WriteString(title)
	content.WriteString("\n\n")

	// Retro-style tabs with classic brackets
	tabs := []string{"[NODES]", "[MODELS]", "[INFERENCE]"}
	var tabContent strings.Builder
	for i, tab := range tabs {
		if Tab(i) == m.currentTab {
			tabContent.WriteString(activeTabStyle.Render(tab))
		} else {
			tabContent.WriteString(inactiveTabStyle.Render(tab))
		}
		tabContent.WriteString(" ")
	}
	content.WriteString(tabContent.String())
	content.WriteString("\n\n")

	// Separator line
	separatorStyle := lipgloss.NewStyle().Foreground(darkGreen)
	separator := strings.Repeat("═", min(m.width-4, 80))
	content.WriteString(separatorStyle.Render(separator))
	content.WriteString("\n\n")

	// Tab content
	switch m.currentTab {
	case TabNodes:
		content.WriteString(m.renderNodesTab())
	case TabModels:
		content.WriteString(m.renderModelsTab())
	case TabInference:
		content.WriteString(m.renderInferenceTab())
	}

	// Footer with retro styling
	content.WriteString("\n")
	footerSep := strings.Repeat("─", min(m.width-4, 80))
	content.WriteString(footerStyle.Render(footerSep))
	content.WriteString("\n")
	content.WriteString(footerStyle.Render("F1=HELP │ TAB=SWITCH │ ↑↓=NAVIGATE │ Q=QUIT"))
	content.WriteString(footerStyle.Render(fmt.Sprintf(" │ UPTIME: %s", m.lastUpdate.Format("15:04:05"))))

	return content.String()
}

// Helper function for minimum value
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m Model) renderNodesTab() string {
	var content strings.Builder

	// Retro header with ASCII decoration
	content.WriteString(headerStyle.Render("▓▓▓ CLUSTER NODES STATUS ▓▓▓"))
	content.WriteString("\n\n")

	if len(m.nodes) == 0 {
		noNodesMsg := `
   ┌─────────────────────────────────────┐
   │  NO NODES DETECTED IN CLUSTER      │
   │  WAITING FOR AGENT CONNECTIONS...  │
   └─────────────────────────────────────┘`
		content.WriteString(asciiStyle.Render(noNodesMsg))
		return content.String()
	}

	// Node counter
	nodeCountStyle := lipgloss.NewStyle().Foreground(darkGreen)
	content.WriteString(nodeCountStyle.Render(fmt.Sprintf("ACTIVE NODES: %d", len(m.nodes))))
	content.WriteString("\n\n")

	for i, node := range m.nodes {
		nodeContent := m.renderNode(node, i == m.selectedNode)
		content.WriteString(nodeContent)
		content.WriteString("\n")
	}

	return content.String()
}

func (m Model) renderNode(node models.Node, selected bool) string {
	style := nodeStyle
	if selected {
		style = selectedNodeStyle
	}

	var statusIcon string
	var statusStyle lipgloss.Style
	switch node.Status {
	case models.NodeStatusOnline:
		statusIcon = "█ ONLINE "
		statusStyle = statusOnlineStyle
	case models.NodeStatusOffline:
		statusIcon = "░ OFFLINE"
		statusStyle = statusOfflineStyle
	case models.NodeStatusBusy:
		statusIcon = "▓ BUSY   "
		statusStyle = statusBusyStyle
	}

	// Create retro-styled node display
	nodeHeader := fmt.Sprintf("NODE: %s", strings.ToUpper(node.ID))
	content := headerStyle.Render(nodeHeader) + "\n"
	content += statusStyle.Render(statusIcon) + "\n"
	content += fmt.Sprintf("ADDR: %s:%d\n", node.Address, node.Port)
	content += fmt.Sprintf("CPU:  %d CORES │ RAM: %d MB\n", node.Resources.CPUCores, node.Resources.MemoryMB)

	if len(node.Resources.GPUs) > 0 {
		content += "GPU:  "
		for i, gpu := range node.Resources.GPUs {
			if i > 0 {
				content += " │ "
			}
			content += fmt.Sprintf("%s (%dMB)", strings.ToUpper(gpu.Name), gpu.MemoryMB)
		}
		content += "\n"
	}

	layersUsed := node.Resources.UsedLayers
	layersMax := node.Resources.MaxLayers
	layersAvail := layersMax - layersUsed

	// Create a visual bar for layer usage
	barWidth := 20

	// Ensure we don't divide by zero and handle edge cases
	if layersMax <= 0 {
		layersMax = 1 // Default to prevent division by zero
	}

	usedBars := int(float64(layersUsed) / float64(layersMax) * float64(barWidth))

	// Ensure usedBars doesn't exceed barWidth
	if usedBars > barWidth {
		usedBars = barWidth
	}
	if usedBars < 0 {
		usedBars = 0
	}

	availBars := barWidth - usedBars

	// Ensure availBars is not negative
	if availBars < 0 {
		availBars = 0
	}

	layerBar := strings.Repeat("█", usedBars) + strings.Repeat("░", availBars)
	content += fmt.Sprintf("LYRS: [%s] %d/%d AVAIL", layerBar, layersAvail, layersMax)

	return style.Render(content)
}

func (m Model) renderModelsTab() string {
	var content strings.Builder

	content.WriteString(headerStyle.Render("▓▓▓ MODEL REPOSITORY ▓▓▓"))
	content.WriteString("\n\n")

	if len(m.modelList) == 0 {
		noModelsMsg := `
   ┌─────────────────────────────────────┐
   │  NO MODELS UPLOADED TO CLUSTER     │
   │  USE UPLOAD COMMAND TO ADD MODELS  │
   └─────────────────────────────────────┘`
		content.WriteString(asciiStyle.Render(noModelsMsg))
		return content.String()
	}

	// Model counter
	modelCountStyle := lipgloss.NewStyle().Foreground(darkGreen)
	content.WriteString(modelCountStyle.Render(fmt.Sprintf("LOADED MODELS: %d", len(m.modelList))))
	content.WriteString("\n\n")

	for _, model := range m.modelList {
		modelContent := fmt.Sprintf("MODEL: %s (v%s)\n", strings.ToUpper(model.Name), model.Version)
		modelContent += fmt.Sprintf("ID:    %s\n", model.ID)
		modelContent += fmt.Sprintf("LYRS:  %d │ SIZE: %d MB\n", model.LayerCount, model.Size/1024/1024)
		modelContent += fmt.Sprintf("PATH:  %s", model.FilePath)

		content.WriteString(nodeStyle.Render(modelContent))
		content.WriteString("\n")
	}

	return content.String()
}

func (m Model) renderInferenceTab() string {
	var content strings.Builder

	content.WriteString(headerStyle.Render("▓▓▓ INFERENCE CONSOLE ▓▓▓"))
	content.WriteString("\n\n")

	// Retro computer art
	computerArt := `
    ┌─────────────────────────────────────────┐
    │ ████████████████████████████████████████│
    │ █                                    █ │
    │ █  DISTRIBUTED INFERENCE ENGINE     █ │
    │ █                                    █ │
    │ █  STATUS: READY FOR DEPLOYMENT     █ │
    │ █                                    █ │
    │ ████████████████████████████████████████│
    └─────────────────────────────────────────┘`

	content.WriteString(asciiStyle.Render(computerArt))
	content.WriteString("\n\n")

	// Feature list with retro styling
	features := []string{
		"► SELECT MODEL FROM REPOSITORY",
		"► ENTER INFERENCE PROMPTS",
		"► VIEW DISTRIBUTED PROCESSING",
		"► MONITOR LAYER DISTRIBUTION",
		"► REAL-TIME PERFORMANCE METRICS",
	}

	content.WriteString(headerStyle.Render("AVAILABLE FEATURES:"))
	content.WriteString("\n\n")

	for _, feature := range features {
		featureStyle := lipgloss.NewStyle().Foreground(paleGreen).PaddingLeft(2)
		content.WriteString(featureStyle.Render(feature))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	comingSoonStyle := lipgloss.NewStyle().
		Foreground(warningAmber).
		Bold(true).
		PaddingLeft(2)
	content.WriteString(comingSoonStyle.Render(">>> IMPLEMENTATION IN PROGRESS <<<"))

	return content.String()
}

func NewModel() Model {
	return Model{
		currentTab:   TabNodes,
		nodes:        []models.Node{},
		modelList:    []models.Model{},
		selectedNode: 0,
		lastUpdate:   time.Now(),
	}
}

func (m *Model) UpdateNodes(nodes []models.Node) {
	m.nodes = nodes
}

func (m *Model) UpdateModels(models []models.Model) {
	m.modelList = models
}
