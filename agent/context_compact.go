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

	if _, err := os.Stat(stored_path); os.IsNotExist(err) {
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

func MicroCompact(messages *[]anthropic.MessageParam) {
	totalToolResults := 0

	// 统计所有 tool_result
	for msgidx := range *messages {
		msg := &(*messages)[msgidx]
		if msg.Role != "user" {
			continue
		}
		for blkidx := range msg.Content {
			block := &msg.Content[blkidx]
			// 安全判断：只统计 ToolResultBlock
			if is_tool_result(block) {
				totalToolResults++
			}
		}
	}

	if totalToolResults <= KEEP_RECENT_TOOL_RESULTS {
		return
	}

	needCompact := totalToolResults - KEEP_RECENT_TOOL_RESULTS
	compacted := 0

	// 正序遍历，压缩最早的
	for msgIdx := range *messages {
		msg := &(*messages)[msgIdx]
		if msg.Role != "user" {
			continue
		}

		for blockIdx := range msg.Content {
			block := &msg.Content[blockIdx]

			// 安全获取 ToolResult
			tr, ok := getToolResult(block)
			if !ok {
				continue
			}
			if len(tr.Content) <= 120 {
				continue
			}

			// 原地压缩
			msg.Content[blockIdx] = anthropic.NewToolResultBlock(
				tr.ToolUseID,
				"[Earlier tool result compacted. Re-run the tool if you need full detail.]",
				false,
			)

			compacted++
			if compacted >= needCompact {
				return
			}
		}
	}
}

// 安全判断是否为 ToolResultBlock
func is_tool_result(block *anthropic.ContentBlockParamUnion) bool {
	return block != nil && block.OfToolResult != nil
}

// 安全获取 ToolResultBlock
func getToolResult(block *anthropic.ContentBlockParamUnion) (anthropic.ToolResultBlockParam, bool) {
	if !is_tool_result(block) {
		return anthropic.ToolResultBlockParam{}, false
	}
	return *block.OfToolResult, true
}
