package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"glcc/agent"
)

// 向上查找 config/config.yaml
func findConfig() string {
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

func main() {
	configPath := findConfig()
	if configPath == "" {
		fmt.Println("Error: could not find config/config.yaml")
		os.Exit(1)
	}

	cfg, err := agent.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Anthropic.APIKey == "" || cfg.Anthropic.APIKey == "your-api-key-here" {
		fmt.Println("Please set your API key in config.yaml")
		os.Exit(1)
	}

	// Ctrl+C 优雅退出
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
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Agent Loop ready (q/exit to quit)")
	for {
		fmt.Print("\033[36ms01 >> \033[0m")
		query, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		query = strings.TrimSpace(query)
		if query == "" || query == "q" || query == "exit" {
			break
		}

		history = append(history, anthropic.NewUserMessage(anthropic.NewTextBlock(query)))
		agent.AgentLoop(&history, client, cfg.Model.ID, system)

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