package tester
import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"glcc/agent"
	"glcc/config"
)

func TestRunSubagent(t *testing.T) {
	// 1. 加载配置 + 创建 client（和你原来一样）
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

	// 2. 直接调用 run_subagent！！！
	// 这里传入一个简单任务，让子代理执行
	result := agent.RunSubagent(
		client,
		cfg.Model.ID,
		`你是一个子代理，请计算 123 + 456 等于多少，只返回结果数字`,
	)

	// 3. 输出结果，看是否正常
	t.Log("子代理返回结果：", result)

	// 4. 简单断言（可选）
	if result == "" || result == "(no summary)" {
		t.Error("子代理没有返回有效内容")
	}
}