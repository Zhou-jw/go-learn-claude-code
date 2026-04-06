package agent

import "github.com/anthropics/anthropic-sdk-go"

const KEEP_RECENT = 3

func Micro_compact(messages []anthropic.MessageParam) {
	var tool_results []anthropic.ContentBlockUnion
	for i, msg := range messages {
		if msg.Role == "user" && len(msg.Content) > 0 {
			for j, block := range msg.Content {
				switch block.(type) {
				case anthropic.ToolUseBlock:
					tool := block.(anthropic.ToolUseBlock)
				}
			}
		}
		
	}
}
