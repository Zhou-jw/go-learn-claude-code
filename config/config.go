package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Anthropic AnthropicConfig `yaml:"anthropic"`
	Model     ModelConfig     `yaml:"model"`
}

type AnthropicConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

type ModelConfig struct {
	ID string `yaml:"id"`
}

func FindConfig() string {
	dir, _ := os.Getwd()
	for {
		testPath := filepath.Join(dir, "config", "config.yaml")
		if _, err := os.Stat(testPath); err == nil {
			return testPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}