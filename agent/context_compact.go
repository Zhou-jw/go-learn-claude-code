package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"glcc/agent/utils"
	"os"
	"path/filepath"
	"strconv"
	"time"

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

// type CompactState struct {
// 	HasCompacted bool
// 	LastSummary  string
// 	RecentFiles  []string
// }

func init_context_compact() {
	TOOL_RESULTS_DIR = filepath.Join(utils.PERSIST_DIR, "tool_results")
	TRANSCRIPT_DIR = filepath.Join(utils.PERSIST_DIR, "transcripts")
	err := os.MkdirAll(TOOL_RESULTS_DIR, 0755)
	if err != nil {
		fmt.Printf("Failed to save: %v\n", err)
	}

	err = os.MkdirAll(TRANSCRIPT_DIR, 0755)
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

// -- Layer 1: micro_compact - replace old tool results with placeholders --
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

// -- Layer 2: auto_compact - save transcript, summarize, replace messages --

/*
 * save the full transcript to disk 
 * return the path to the saved file
 */
func WriteTranscript(messages *[]anthropic.MessageParam) (string, error) {
	// Save full transcript to disk
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	path := filepath.Join(TRANSCRIPT_DIR, "transcript_"+ts+".jsonl")

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, msg := range *messages {
		if err := encoder.Encode(msg); err != nil {
			return path, err
		}
	}

	return path, nil
}

func SummarizeHistory(messages *[]anthropic.MessageParam, r *anthropic.Client, modelID string) (*anthropic.Message, error) {
	conv_bytes, _ := json.Marshal(messages)
	conversation := string(conv_bytes)
	if len(conversation) > 80000 {
		conversation = conversation[len(conversation)-80000:]
	}

	sum_prompt := fmt.Sprintf(`Summarize this coding-agent conversation so work can continue.
		Preserve:
		1. The current goal
		2. Important findings and decisions
		3. Files read or changed
		4. Remaining work
		5. User constraints and preferences
		Be compact but concrete.
		%s`, conversation)

	resp, err := r.Messages.New(context.TODO(), anthropic.MessageNewParams{
		Model:     anthropic.Model(modelID),
		MaxTokens: 2000,
		System: []anthropic.TextBlockParam{
			{Text: sum_prompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(sum_prompt)),
		},
		Tools: PARENT_TOOLS,
	})
	if err != nil {
		return &anthropic.Message{}, err
	}

	if len(resp.Content) == 0 {
		return &anthropic.Message{}, fmt.Errorf("empty summary response")
	}

	return resp, nil
}

func auto_compact(messages *[]anthropic.MessageParam, r *anthropic.Client, modelID string, /*state *CompactState*/) (*[]anthropic.MessageParam, error) {
	transcriptPath, err := WriteTranscript(messages)
	if err == nil {
		fmt.Printf("[transcript saved: %s]\n", transcriptPath)
	}

	summaryMsg, err := SummarizeHistory(messages, r, modelID)
	if err != nil {
		return nil, err
	}

	var summary string
	if len(summaryMsg.Content) > 0 {
		summary = summaryMsg.Content[0].Text
	}
	if summary == "" {
		summary = "No summary generated."
	}

	// if len(state.RecentFiles) > 0 {
	// 	summary += "\n\nRecent files to reopen if needed:\n"
	// 	for _, f := range state.RecentFiles {
	// 		summary += "- " + f + "\n"
	// 	}
	// }

	// state.HasCompacted = true
	// state.LastSummary = summary

	compressedContent := fmt.Sprintf("[Conversation compressed. Transcript: %s]\n\n%s", transcriptPath, summary)
	msg := anthropic.NewUserMessage(anthropic.NewTextBlock(compressedContent))
	return &[]anthropic.MessageParam{msg}, nil
}

func estimated_tokens(messages *[]anthropic.MessageParam/* , r *anthropic.Client, modelID string*/) int {
	// msg_tokens_cnt, err := r.Messages.CountTokens(context.TODO(), anthropic.MessageCountTokensParams{
	// 	Model:    modelID,
	// 	Messages: *messages,
	// })
	// if err != nil {
	// 	panic(err.Error())
	// }
	
	conv_bytes, _ := json.Marshal(messages)
	conversation := string(conv_bytes)
	msg_tokens_cnt := len(conversation) / 4
	
	return int(msg_tokens_cnt)
}