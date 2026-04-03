package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

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
			Tools:    TOOLS,
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

				fmt.Printf("\033[33m> %s\033[0m\n", toolUse.Name)
				output := DispatchTool(toolUse.Name, input)
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
