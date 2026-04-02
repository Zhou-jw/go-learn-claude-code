package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
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

var dangerousCommands = []string{
	"rm -rf /",
	"sudo",
	"shutdown",
	"reboot",
	"> /dev/",
}

func runBash(command string) string {
	for _, d := range dangerousCommands {
		if strings.Contains(command, d) {
			return "Error: Dangerous command blocked"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir, _ = os.Getwd()

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "Error: Timeout (120s)"
	}
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	out := strings.TrimSpace(string(output))
	if out == "" {
		return "(no output)"
	}
	if len(out) > 50000 {
		return out[:50000]
	}
	return out
}

func AgentLoop(messages *[]anthropic.MessageParam, client anthropic.Client, modelID string, system string) {
	if modelID == "" {
		modelID = string(anthropic.ModelClaudeSonnet4_20250514)
	}

	for {
		resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(modelID),
			MaxTokens: 8000,
			System: []anthropic.TextBlockParam{
				{Text: system},
			},
			Messages: *messages,
			Tools: []anthropic.ToolUnionParam{
				anthropic.ToolUnionParamOfTool(
					anthropic.ToolInputSchemaParam{
						Type: "object",
						Properties: map[string]any{
							"command": map[string]any{
								"type":        "string",
								"description": "The bash command to execute",
							},
						},
						Required: []string{"command"},
					},
					"bash",
				),
			},
		})
		if err != nil {
			fmt.Printf("API Error: %v\n", err)
			return
		}

		var assistantContent []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			switch b := block.AsAny().(type) {
			case anthropic.TextBlock:
				assistantContent = append(assistantContent, anthropic.NewTextBlock(b.Text))
			case anthropic.ToolUseBlock:
				inputJSON, _ := json.Marshal(b.Input)
				assistantContent = append(assistantContent, anthropic.NewToolUseBlock(b.ID, json.RawMessage(inputJSON), b.Name))
			}
		}
		*messages = append(*messages, anthropic.NewAssistantMessage(assistantContent...))

		if resp.StopReason != anthropic.StopReasonToolUse {
			return
		}

		var toolResults []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			if toolUse, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				var input map[string]any
				json.Unmarshal(toolUse.Input, &input)
				cmd := input["command"].(string)
				fmt.Printf("\033[33m$ %s\033[0m\n", cmd)
				output := runBash(cmd)
				if len(output) > 200 {
					fmt.Println(output[:200] + "...")
				} else {
					fmt.Println(output)
				}
				toolResults = append(toolResults, anthropic.NewToolResultBlock(toolUse.ID, output, false))
			}
		}

		*messages = append(*messages, anthropic.NewUserMessage(toolResults...))
	}
}
