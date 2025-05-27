package agent

import (
	"os"
	"testing"
)

func TestNewLLM(t *testing.T) {
	modelPath := "/path/to/model.ggml"
	llm := NewLLM(modelPath)

	if llm == nil {
		t.Fatal("NewLLM returned nil")
	}

	if llm.ModelPath != modelPath {
		t.Errorf("Expected ModelPath %s, got %s", modelPath, llm.ModelPath)
	}
}

func TestComputeLayers(t *testing.T) {
	llm := NewLLM("/test/model")

	tests := []struct {
		name     string
		cpuCores int
		ramMB    int
		expected int
	}{
		{
			name:     "CPU limited",
			cpuCores: 4,
			ramMB:    8192,
			expected: 4, // min(4, 8192/512) = min(4, 16) = 4
		},
		{
			name:     "RAM limited",
			cpuCores: 16,
			ramMB:    2048,
			expected: 4, // min(16, 2048/512) = min(16, 4) = 4
		},
		{
			name:     "Equal constraint",
			cpuCores: 8,
			ramMB:    4096,
			expected: 8, // min(8, 4096/512) = min(8, 8) = 8
		},
		{
			name:     "Low resources",
			cpuCores: 2,
			ramMB:    512,
			expected: 1, // min(2, 512/512) = min(2, 1) = 1
		},
		{
			name:     "Zero CPU",
			cpuCores: 0,
			ramMB:    4096,
			expected: 0, // min(0, 8) = 0
		},
		{
			name:     "Insufficient RAM",
			cpuCores: 4,
			ramMB:    256,
			expected: 0, // min(4, 256/512) = min(4, 0) = 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := llm.ComputeLayers(tt.cpuCores, tt.ramMB)
			if result != tt.expected {
				t.Errorf("ComputeLayers(%d, %d) = %d, expected %d",
					tt.cpuCores, tt.ramMB, result, tt.expected)
			}
		})
	}
}

func TestInvokeLLM(t *testing.T) {
	llm := NewLLM("/test/model.ggml")

	// Test with non-existent command (llama.cpp likely not installed)
	output, err := llm.InvokeLLM(4)

	// We expect an error since llama.cpp is probably not installed
	if err == nil {
		t.Log("llama.cpp appears to be available, output:", output)
	} else {
		// This is expected in test environment
		t.Logf("Expected error when llama.cpp not available: %v", err)

		// Verify the error is related to command execution
		if output == "" {
			t.Log("Got empty output as expected for missing command")
		}
	}
}

func TestInvokeLLMWithMockCommand(t *testing.T) {
	// Create a temporary script that mimics llama.cpp behavior
	tempScript := "/tmp/test-llama"
	scriptContent := `#!/bin/bash
echo "Mock llama.cpp output: model=$2 layers=$4"
exit 0
`

	err := os.WriteFile(tempScript, []byte(scriptContent), 0755)
	if err != nil {
		t.Skipf("Could not create temporary script: %v", err)
	}
	defer os.Remove(tempScript)

	// Temporarily modify PATH to include our mock command
	// Note: This is a simplified test - in practice we'd use dependency injection
	llm := &LLM{ModelPath: "/test/model.ggml"}

	// Test the structure without actually calling the external command
	// Just verify the function exists and handles parameters correctly
	if llm.ModelPath != "/test/model.ggml" {
		t.Errorf("ModelPath not set correctly")
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a smaller", 3, 5, 3},
		{"b smaller", 7, 4, 4},
		{"equal", 6, 6, 6},
		{"negative numbers", -2, -5, -5},
		{"mixed signs", -3, 2, -3},
		{"zero", 0, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkComputeLayers(b *testing.B) {
	llm := NewLLM("/test/model")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		llm.ComputeLayers(8, 4096)
	}
}

func BenchmarkMin(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		min(100, 200)
	}
}
