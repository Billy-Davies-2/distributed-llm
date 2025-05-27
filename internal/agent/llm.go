package agent

import (
    "os/exec"
    "strconv"
)

// LLM represents the structure for managing interactions with the LLM.
type LLM struct {
    ModelPath string
}

// NewLLM creates a new instance of LLM with the specified model path.
func NewLLM(modelPath string) *LLM {
    return &LLM{ModelPath: modelPath}
}

// ComputeLayers calculates how many layers of the LLM can be run based on available resources.
func (l *LLM) ComputeLayers(cpuCores int, ramMB int) int {
    // Example logic: Assume each layer requires 1 core and 512MB of RAM
    layersByCPU := cpuCores
    layersByRAM := ramMB / 512
    return min(layersByCPU, layersByRAM)
}

// InvokeLLM runs the llama.cpp executable with the specified parameters.
func (l *LLM) InvokeLLM(layers int) (string, error) {
    cmd := exec.Command("llama.cpp", "--model", l.ModelPath, "--layers", strconv.Itoa(layers))
    output, err := cmd.CombinedOutput()
    return string(output), err
}

// min returns the smaller of two integers.
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}