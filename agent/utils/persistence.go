package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

var (
	PERSIST_DIR  = "agent_logs"
	PERSIST_FILE = ""
	saveLock     sync.Mutex // 并发安全锁，防止多协程覆盖
)

func Init_persist() {
	PERSIST_DIR = fmt.Sprintf("%s/%s", PERSIST_DIR, time.Now().Format("20060102_150405"))
	PERSIST_FILE = fmt.Sprintf("%s/messages.jsonl", PERSIST_DIR)
}

type PersistedMessage struct {
	Timestamp string                   `json:"timestamp"`
	Round     int                      `json:"round"`
	Type      string                   `json:"type"` // "parent" or "subagent"
	Messages  []anthropic.MessageParam `json:"messages"`
}

type PersistedSession struct {
	Sessions []PersistedMessage `json:"sessions"`
}

func SaveMessages(messages *[]anthropic.MessageParam, agent_name string, round int, msgType string) error {
	saveLock.Lock()
	defer saveLock.Unlock()

	// 创建目录（必须检查错误）
	if err := os.MkdirAll(PERSIST_DIR, 0755); err != nil {
		return err
	}

	msg := PersistedMessage{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Round:     round,
		Type:      msgType,
		Messages:  *messages,
	}
	
	var file_path string
	if agent_name == "" {
		file_path = PERSIST_FILE
	} else {
		file_path = fmt.Sprintf("%s/%s_messages.jsonl", PERSIST_DIR, agent_name)
	}
	
	file, err := os.OpenFile(file_path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// 格式化写入
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(msg)
}
