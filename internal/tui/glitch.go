package tui

import (
	"math/rand"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// GlitchEffect handles retro terminal glitches
type GlitchEffect struct {
	enabled     bool
	intensity   float64
	lastGlitch  time.Time
	glitchChars []rune
	glitchStyle lipgloss.Style
}

// NewGlitchEffect creates a new glitch effect system
func NewGlitchEffect() *GlitchEffect {
	return &GlitchEffect{
		enabled:   true,
		intensity: 0.15, // 15% chance of glitches
		glitchChars: []rune{
			'░', '▒', '▓', '█', '▄', '▀', '▌', '▐',
			'┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼',
			'╔', '╗', '╚', '╝', '╠', '╣', '╦', '╩', '╬',
			'■', '□', '▪', '▫', '●', '○', '◆', '◇',
			'¤', '‡', '†', '§', '¶', '©', '®', '™',
			'α', 'β', 'γ', 'δ', 'ε', 'ζ', 'η', 'θ',
		},
		glitchStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Background(lipgloss.Color("#00FF00")).
			Blink(true),
	}
}

// ShouldGlitch determines if a glitch should occur
func (g *GlitchEffect) ShouldGlitch() bool {
	if !g.enabled {
		return false
	}

	// Increase glitch probability over time since last glitch
	timeSinceGlitch := time.Since(g.lastGlitch)
	baseChance := g.intensity

	// Increase chance gradually, but cap it
	timeBonus := float64(timeSinceGlitch.Seconds()) * 0.001
	totalChance := baseChance + timeBonus
	if totalChance > 0.3 {
		totalChance = 0.3
	}

	if rand.Float64() < totalChance {
		g.lastGlitch = time.Now()
		return true
	}

	return false
}

// ApplyTextGlitch applies various text glitches
func (g *GlitchEffect) ApplyTextGlitch(text string) string {
	if !g.ShouldGlitch() {
		return text
	}

	glitchType := rand.Intn(8)

	switch glitchType {
	case 0:
		return g.corruptCharacters(text)
	case 1:
		return g.addStaticLines(text)
	case 2:
		return g.duplicateLines(text)
	case 3:
		return g.scrambleText(text)
	case 4:
		return g.addNoise(text)
	case 5:
		return g.invertColors(text)
	case 6:
		return g.addFlicker(text)
	case 7:
		return g.addScanlines(text)
	default:
		return text
	}
}

// corruptCharacters randomly replaces characters with glitch characters
func (g *GlitchEffect) corruptCharacters(text string) string {
	if len(text) == 0 {
		return text
	}

	runes := []rune(text)
	numToCorrupt := rand.Intn(len(runes)/10 + 1)

	for i := 0; i < numToCorrupt; i++ {
		pos := rand.Intn(len(runes))
		if !unicode.IsSpace(runes[pos]) {
			runes[pos] = g.glitchChars[rand.Intn(len(g.glitchChars))]
		}
	}

	return string(runes)
}

// addStaticLines adds random static lines
func (g *GlitchEffect) addStaticLines(text string) string {
	lines := strings.Split(text, "\n")

	// Add 1-3 static lines at random positions
	numLines := rand.Intn(3) + 1
	for i := 0; i < numLines; i++ {
		staticLine := g.generateStaticLine(60 + rand.Intn(20))
		pos := rand.Intn(len(lines) + 1)

		// Insert static line
		lines = append(lines[:pos], append([]string{staticLine}, lines[pos:]...)...)
	}

	return strings.Join(lines, "\n")
}

// generateStaticLine creates a line of static
func (g *GlitchEffect) generateStaticLine(length int) string {
	static := make([]rune, length)
	staticChars := []rune{'░', '▒', '▓', '█', '▄', '▀', ' '}

	for i := range static {
		static[i] = staticChars[rand.Intn(len(staticChars))]
	}

	return g.glitchStyle.Render(string(static))
}

// duplicateLines randomly duplicates some lines
func (g *GlitchEffect) duplicateLines(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return text
	}

	// Duplicate 1-2 random lines
	numDupes := rand.Intn(2) + 1
	for i := 0; i < numDupes; i++ {
		lineIndex := rand.Intn(len(lines))
		duplicatedLine := g.addLineGlitch(lines[lineIndex])

		// Insert duplicate near original
		insertPos := lineIndex + 1
		if insertPos > len(lines) {
			insertPos = len(lines)
		}

		lines = append(lines[:insertPos], append([]string{duplicatedLine}, lines[insertPos:]...)...)
	}

	return strings.Join(lines, "\n")
}

// addLineGlitch adds glitch effects to a single line
func (g *GlitchEffect) addLineGlitch(line string) string {
	// Add some corruption
	corrupted := g.corruptCharacters(line)

	// Maybe add color inversion
	if rand.Float64() < 0.3 {
		corrupted = g.glitchStyle.Render(corrupted)
	}

	return corrupted
}

// scrambleText randomly scrambles parts of the text
func (g *GlitchEffect) scrambleText(text string) string {
	runes := []rune(text)
	if len(runes) < 4 {
		return text
	}

	// Scramble 2-4 small sections
	numSections := rand.Intn(3) + 2
	for i := 0; i < numSections; i++ {
		start := rand.Intn(len(runes) - 3)
		length := rand.Intn(4) + 2
		end := start + length
		if end > len(runes) {
			end = len(runes)
		}

		// Shuffle this section
		section := runes[start:end]
		rand.Shuffle(len(section), func(i, j int) {
			section[i], section[j] = section[j], section[i]
		})
	}

	return string(runes)
}

// addNoise adds random noise characters
func (g *GlitchEffect) addNoise(text string) string {
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if rand.Float64() < 0.2 { // 20% chance per line
			// Add noise at random position
			noisePos := rand.Intn(len(line) + 1)
			noise := string(g.glitchChars[rand.Intn(len(g.glitchChars))])

			if rand.Float64() < 0.5 {
				noise = g.glitchStyle.Render(noise)
			}

			lines[i] = line[:noisePos] + noise + line[noisePos:]
		}
	}

	return strings.Join(lines, "\n")
}

// invertColors inverts colors of random sections
func (g *GlitchEffect) invertColors(text string) string {
	// This is a simple implementation - in a real terminal this would invert fg/bg
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if rand.Float64() < 0.1 { // 10% chance per line
			lines[i] = g.glitchStyle.Render(line)
		}
	}

	return strings.Join(lines, "\n")
}

// addFlicker simulates screen flicker by occasionally replacing text
func (g *GlitchEffect) addFlicker(text string) string {
	if rand.Float64() < 0.05 { // 5% chance
		// Replace entire text with flicker pattern
		flickerPattern := strings.Repeat("█", len(text)/2) + strings.Repeat(" ", len(text)/2)
		return g.glitchStyle.Render(flickerPattern)
	}
	return text
}

// addScanlines adds horizontal scanline effects
func (g *GlitchEffect) addScanlines(text string) string {
	lines := strings.Split(text, "\n")

	// Add scanlines every few lines
	for i := 0; i < len(lines); i += rand.Intn(4) + 3 {
		if i < len(lines) {
			scanline := strings.Repeat("─", 80)
			scanlineStyled := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#004400")).
				Render(scanline)

			lines = append(lines[:i], append([]string{scanlineStyled}, lines[i:]...)...)
		}
	}

	return strings.Join(lines, "\n")
}

// ApplyScreenGlitch applies screen-wide glitch effects
func (g *GlitchEffect) ApplyScreenGlitch() (bool, string) {
	if !g.ShouldGlitch() {
		return false, ""
	}

	glitchType := rand.Intn(4)

	switch glitchType {
	case 0:
		// Full screen static
		return true, g.generateFullScreenStatic()
	case 1:
		// Screen flicker (empty screen)
		return true, ""
	case 2:
		// Horizontal bars
		return true, g.generateHorizontalBars()
	case 3:
		// Vertical interference
		return true, g.generateVerticalInterference()
	default:
		return false, ""
	}
}

// generateFullScreenStatic creates full screen of static
func (g *GlitchEffect) generateFullScreenStatic() string {
	lines := make([]string, 20)
	staticChars := []rune{'░', '▒', '▓', '█', '▄', '▀', ' ', '·', '∴', '∵'}

	for i := range lines {
		line := make([]rune, 80)
		for j := range line {
			line[j] = staticChars[rand.Intn(len(staticChars))]
		}
		lines[i] = string(line)
	}

	return strings.Join(lines, "\n")
}

// generateHorizontalBars creates horizontal interference bars
func (g *GlitchEffect) generateHorizontalBars() string {
	lines := make([]string, 20)

	for i := range lines {
		if i%3 == 0 {
			lines[i] = strings.Repeat("█", 80)
		} else {
			lines[i] = strings.Repeat(" ", 80)
		}
	}

	return g.glitchStyle.Render(strings.Join(lines, "\n"))
}

// generateVerticalInterference creates vertical interference pattern
func (g *GlitchEffect) generateVerticalInterference() string {
	lines := make([]string, 20)

	for i := range lines {
		line := make([]rune, 80)
		for j := range line {
			if j%4 == 0 {
				line[j] = '▐'
			} else if j%4 == 2 {
				line[j] = '▌'
			} else {
				line[j] = ' '
			}
		}
		lines[i] = string(line)
	}

	return strings.Join(lines, "\n")
}

// SetIntensity adjusts the glitch intensity (0.0 to 1.0)
func (g *GlitchEffect) SetIntensity(intensity float64) {
	if intensity < 0 {
		intensity = 0
	}
	if intensity > 1 {
		intensity = 1
	}
	g.intensity = intensity
}

// SetEnabled toggles glitch effects on/off
func (g *GlitchEffect) SetEnabled(enabled bool) {
	g.enabled = enabled
}

// IsEnabled returns whether glitch effects are enabled
func (g *GlitchEffect) IsEnabled() bool {
	return g.enabled
}
