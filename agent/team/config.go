package team

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Member struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

type TeamConfig struct {
	TeamName string   `json:"team_name"`
	Members  []Member `json:"members"`
}

func LoadTeamConfig(teamDir string) TeamConfig {
	configPath := filepath.Join(teamDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return TeamConfig{TeamName: "default", Members: []Member{}}
	}
	var cfg TeamConfig
	json.Unmarshal(data, &cfg)
	return cfg
}

func SaveTeamConfig(teamDir string, cfg TeamConfig) {
	configPath := filepath.Join(teamDir, "config.json")
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath, data, 0644)
}

func FindMember(cfg *TeamConfig, name string) *Member {
	for i := range cfg.Members {
		if cfg.Members[i].Name == name {
			return &cfg.Members[i]
		}
	}
	return nil
}

func shutdown_member(cfg *TeamConfig, name string) {
	member := FindMember(cfg, name)
	if member == nil {
		return
	}
	member.Status = "shutdown"
}