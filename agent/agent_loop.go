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

	round := 0
	for {
		// layer1: micro_compact before each LLM call
		MicroCompact(messages)

		// layer2: auto_compact if token estimate exceeds threshold
		if estimated_tokens(messages) > COMPACT_THRESHOLD {
			var err error
			fmt.Print("\033[33m> layer2: auto_compact triggered\033[0m\n")
			messages, err = auto_compact(messages, &client, modelID)
			if err != nil {
				fmt.Printf("auto_compact error: %v\n", err)
				return
			}
		}

		resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(modelID),
			MaxTokens: 8000,
			System: []anthropic.TextBlockParam{
				{Text: system},
			},
			Messages: *messages,
			Tools:    PARENT_TOOLS,
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

		round++
		SaveMessages(messages, round, "parent")
		
		var toolResults []anthropic.ContentBlockParamUnion
		var need_manual_compact = false
		for _, block := range resp.Content {
			if toolUse, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				var input map[string]any
				json.Unmarshal(toolUse.Input, &input)

				fmt.Printf("\033[33m> %s\033[0m\n", toolUse.Name)
				var output string
				switch toolUse.Name {
				case "task":
					prompt := input["prompt"].(string)
					fmt.Println("> task: ", prompt[:80])
					output = RunSubagent(client, modelID, prompt, round)
				case "compact":
					need_manual_compact = true
					fmt.Println("Compressing...")
				default:
					output = DispatchTool(toolUse.Name, input)
				}

				if len(output) > 200 {
					fmt.Println(output[:200] + "...")
				} else {
					fmt.Println(output)
				}

				toolResults = append(toolResults, anthropic.NewToolResultBlock(toolUse.ID, output, false))
			}

			// layer3 :manual compact triggered by the compact tool
			if need_manual_compact {
				fmt.Print("\033[33m> layer3: manual compact\033[0m\n")
				messages, err = auto_compact(messages, &client, modelID)
				if err != nil {
					fmt.Printf("auto_compact error: %v\n", err)
					return
				}
			}
		}

		*messages = append(*messages, anthropic.NewUserMessage(toolResults...))
	}
}
