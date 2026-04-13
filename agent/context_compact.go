package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/anthropic-sdk-go"
)

const (
	PERSIST_THRESHOLD        = 1024 // 输出超过此长度则持久化
	PREVIEW_CHARS            = 500  // 预览截取长度
	KEEP_RECENT_TOOL_RESULTS = 3    // 保留最近的工具结果数量
)

var (
	TOOL_RESULTS_DIR string
	TRANSCRIPT_DIR   string
)

type CompactState struct {
	HasCompacted bool
	LastSummary  string
	RecentFiles  []string
}

func init() {
	cwd, _ := os.Getwd()
	TOOL_RESULTS_DIR = filepath.Join(cwd, "tool_results")
	TRANSCRIPT_DIR = filepath.Join(cwd, "transcripts")
	err := os.MkdirAll(TOOL_RESULTS_DIR, 0755)
	if err != nil {
		fmt.Printf("Failed to save: %v\n", err)
	}

}

func Persist_large_output(tool_use_id string, output string) string {
	if len(output) <= PERSIST_THRESHOLD {
		return output
	}
	stored_path := filepath.Join(TOOL_RESULTS_DIR, tool_use_id+"txt")
	
	if _, err := os.Stat(stored_path); os.IsNotExist(err){
		err = os.WriteFile(stored_path, []byte(output), 0644)
		if err != nil {
			return fmt.Sprintf("Failed to write: %v", err)
		}
	}
	
	preview := output
	if len(output) > PREVIEW_CHARS {
		preview = output[:PREVIEW_CHARS] + "..."
	}
	
	rel_path, err := filepath.Rel(WORKDIR, stored_path)
	if err != nil {
		rel_path = stored_path
	}
	
	return fmt.Sprintf("Output saved to %s. Preview: %s", rel_path, preview)

}

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
