package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"glcc/agent"
	"glcc/config"
	"glcc/console"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func main() {
	configPath := config.FindConfig()
	if configPath == "" {
		console.Red("Error: could not find config/config.yaml")
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Anthropic.APIKey == "" || cfg.Anthropic.APIKey == "your-api-key-here" {
		console.Red("Please set your API key in config.yaml")
		os.Exit(1)
	}

	// Ctrl+C 优雅退出
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		console.Info("\nExiting...")
		os.Exit(0)
	}()

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.Anthropic.APIKey),
	}
	if cfg.Anthropic.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.Anthropic.BaseURL))
	}
	client := anthropic.NewClient(opts...)

	var history []anthropic.MessageParam
	reader := bufio.NewReader(os.Stdin)

	console.Info("Agent Loop ready (q/exit to quit)")
	agent.InitAll()
	for {
		console.Cyan("s01 >> ")

		query, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		query = strings.TrimSpace(query)
		query = strings.ToLower(query)

		if query == "q" || query == "exit" {
			break
		}
		if query == "/team" {
			console.Info("%s", team.TEAMMATE_MGR.ListAll())
			continue
		}
		if query == "/inbox" {
			continue
		}

		history = append(history, anthropic.NewUserMessage(anthropic.NewTextBlock(query)))
		agent.AgentLoop(&history, client, cfg.Model.ID, agent.Sys_prompt())

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
