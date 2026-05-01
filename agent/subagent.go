package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"glcc/agent/tools"
	"glcc/console"

	"github.com/anthropics/anthropic-sdk-go"
)

func RunSubagent(client anthropic.Client, modelID string, prompt string, parentRound int, agt *Agent) string {
	var sub_messages = []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))}
	if modelID == "" {
		return "modelID not specified"
	}

	round := 0

	var resp *anthropic.Message
	var err error
	for range 30 {
		resp, err = client.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(modelID),
			MaxTokens: 8000,
			System: []anthropic.TextBlockParam{
				{Text: Subagent_sys_prompt()},
			},
			Messages: sub_messages,
			Tools:    SubagentTools(),
		})
		if err != nil {
			output := fmt.Sprintf("API Error: %v\n", err)
			return output
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
		sub_messages = append(sub_messages, anthropic.NewAssistantMessage(assistantContent...))

		if resp.StopReason != anthropic.StopReasonToolUse {
			break
		}

		round++
		agt.Persist.SaveMessages(&sub_messages, "sub_agent", parentRound+round, "subagent")
		var toolResults []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			if toolUse, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				var input map[string]any
				json.Unmarshal(toolUse.Input, &input)

				console.Yellow("> %s\n", toolUse.Name)
				output := agt.Tools.Registry.Dispatch(toolUse.Name, input)
				if len(output) > 50000 {
					output = output[:50000]
				}

				if len(output) > 200 {
					console.Info("%s", output[:200]+"...")
				} else {
					console.Info("%s", output)
				}

				toolResults = append(toolResults, anthropic.NewToolResultBlock(toolUse.ID, output, false))
			}
		}

		sub_messages = append(sub_messages, anthropic.NewUserMessage(toolResults...))
	}

	// return the text of final response
	finalText := ""
	for _, block := range resp.Content {
		if textBlock, ok := block.AsAny().(anthropic.TextBlock); ok {
			finalText += textBlock.Text
		}
	}
	if finalText == "" {
		return "(no summary)"
	}
	return finalText
}

func SubagentTools() []anthropic.ToolUnionParam {
	//(TODO): explore mode: read and write otherwise read
	return tools.CoreTools()
}
