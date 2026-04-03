package tester

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"glcc/agent"
	"glcc/config"
)

func TestAgentLoop(t *testing.T) {
	configPath := config.FindConfig()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		fmt.Println("Please copy example.yaml to config.yaml and fill in your API key")
		os.Exit(1)
	}

	if cfg.Anthropic.APIKey == "" || cfg.Anthropic.APIKey == "your-api-key-here" {
		fmt.Println("Please set your API key in config.yaml")
		os.Exit(1)
	}

	/*
	 * SIGINT: 监听 Ctrl+C 信号，收到时退出程序
	 * SIGTERM: 监听kill ，收到时退出程序
	 */
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
