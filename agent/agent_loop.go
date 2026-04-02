package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
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

func loadConfig(path string) (*Config, error) {
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

func agentLoop(messages *[]anthropic.MessageParam, client anthropic.Client, modelID string, system string) {
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

func TestAgentLoop(t *testing.T) {
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	if execDir == "." {
		execDir, _ = os.Getwd()
	}

	configPath := filepath.Join(execDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = filepath.Join(execDir, "..", "config.yaml")
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "agent/config.yaml"
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		fmt.Println("Please copy example.yaml to config.yaml and fill in your API key")
		os.Exit(1)
	}

	if cfg.Anthropic.APIKey == "" || cfg.Anthropic.APIKey == "your-api-key-here" {
		fmt.Println("Please set your API key in config.yaml")
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\nExiting...")
		os.Exit(0)
	}()

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.Anthropic.APIKey),
	}
	if cfg.Anthropic.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.Anthropic.BaseURL))
	}
	client := anthropic.NewClient(opts...)

	cwd, _ := os.Getwd()
	system := fmt.Sprintf("You are a coding agent at %s. Use bash to solve tasks. Act, don't explain.", cwd)

	var history []anthropic.MessageParam

	fmt.Println("Agent Loop ready (q/exit to quit)")
	for {
		fmt.Print("\033[36ms01 >> \033[0m")
		var query string
		_, err := fmt.Scanln(&query)
		if err != nil {
			break
		}

		query = strings.TrimSpace(query)
		if query == "" || query == "q" || query == "exit" {
			break
		}

		history = append(history, anthropic.NewUserMessage(anthropic.NewTextBlock(query)))

		agentLoop(&history, client, cfg.Model.ID, system)

		if len(history) > 0 {
			lastMsg := history[len(history)-1]
			for _, block := range lastMsg.Content {
				if textBlock := block.GetText(); textBlock != nil && *textBlock != "" {
					fmt.Println(*textBlock)
				}
			}
		}
		fmt.Println()
	}
}
