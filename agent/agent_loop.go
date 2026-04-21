package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"glcc/agent/tools"
	"glcc/console"
	"strings"

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
			console.Cyan("> layer2: auto_compact triggered")
			messages, err = auto_compact(messages, &client, modelID)
			if err != nil {
				console.Red("auto_compact error: %v", err)
				return
			}
		}

		// s08: drain background tasks and append results to messages
		bg_done_tasks := TASKMGR.drain()
		if len(bg_done_tasks) > 0 {
			var lines []string
			for _, bg_task := range bg_done_tasks {
				lines = append(lines, fmt.Sprintf("[bg_task: %d] %s: %s",
					bg_task.ID, bg_task.State, bg_task.Output))
			}
			txt := strings.Join(lines, "\n")
			bg_text := fmt.Sprintf("<background-results>\n%s\n</background-results>", txt)
			*messages = append(*messages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(bg_text),
			))
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
			console.Red("API Error: %v\n", err)
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

				console.Yellow("> %s\n", toolUse.Name)
				var output string
				switch toolUse.Name {
				case "task":
					prompt := input["prompt"].(string)

					console.InfoOfLen(80, "> task: %s", prompt)
					output = RunSubagent(client, modelID, prompt, round)
				case "compact":
					need_manual_compact = true
					console.Info("Compressing...")
				default:
					output = DispatchTool(toolUse.Name, input)
				}

				console.InfoOfLen(200, "%s", output)

				toolResults = append(toolResults, anthropic.NewToolResultBlock(toolUse.ID, output, false))
			}

			// layer3 :manual compact triggered by the compact tool
			if need_manual_compact {
				console.Yellow("> layer3: manual compact\n")
				messages, err = auto_compact(messages, &client, modelID)
				if err != nil {
					console.Red("auto_compact error: %v\n", err)
					return
				}
			}
		}

		*messages = append(*messages, anthropic.NewUserMessage(toolResults...))
	}
}

func MainAgentTools() []anthropic.ToolUnionParam {
	mainagent_tools := SubagentTools()
	mainagent_tools = append(mainagent_tools, tools.SpawnSubagentTool()...)
	return mainagent_tools
}