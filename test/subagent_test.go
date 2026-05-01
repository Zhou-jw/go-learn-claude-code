package tester

import (
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"glcc/agent"
	"glcc/config"
)

func TestRunSubagent(t *testing.T) {
	configPath := config.FindConfig()
	cfg, err := config.LoadConfig(configPath)
	if err != nil || cfg.Anthropic.APIKey == "" {
		t.Fatal("请配置正确的 API Key")
	}

	opts := []option.RequestOption{option.WithAPIKey(cfg.Anthropic.APIKey)}
	if cfg.Anthropic.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.Anthropic.BaseURL))
	}
	client := anthropic.NewClient(opts...)

	cwd, _ := os.Getwd()
	agt := agent.NewAgent(cwd)

	result := agent.RunSubagent(
		client,
		cfg.Model.ID,
		`你是一个子代理，请计算 123 + 456 等于多少，只返回结果数字`,
		0,
		agt,
	)

	t.Log("子代理返回结果：", result)

	if result == "" || result == "(no summary)" {
		t.Error("子代理没有返回有效内容")
	}
}
