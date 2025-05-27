package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	NodeID              string         `json:"node_id"`
	Port                int            `json:"port"`
	GossipPort          int            `json:"gossip_port"`
	KubernetesNamespace string         `json:"kubernetes_namespace"`
	ResourceLimits      ResourceLimits `json:"resource_limits"`
	ModelPath           string         `json:"model_path"`
	DataPath            string         `json:"data_path"`
	LogLevel            string         `json:"log_level"`
}

type ResourceLimits struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	GPU    string `json:"gpu"`
}

func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// Default returns a default configuration
func Default() *Config {
	return &Config{
		NodeID:              "default-node",
		Port:                8080,
		GossipPort:          7946,
		KubernetesNamespace: "default",
		ResourceLimits: ResourceLimits{
			CPU:    "1000m",
			Memory: "2Gi",
			GPU:    "1",
		},
		ModelPath: "/models",
		DataPath:  "/data",
		LogLevel:  "info",
	}
}
