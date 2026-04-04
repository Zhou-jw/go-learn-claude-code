package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

var (
	PERSIST_DIR  = "agent_logs"
	PERSIST_FILE = ""
)

func init() {
	PERSIST_FILE = fmt.Sprintf("%s/messages_%s.json", PERSIST_DIR, time.Now().Format("20060102_150405"))
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

func SaveMessages(messages *[]anthropic.MessageParam, round int, msgType string) error {
	os.MkdirAll(PERSIST_DIR, 0755)

	var session PersistedSession
	if data, err := os.ReadFile(PERSIST_FILE); err == nil {
		json.Unmarshal(data, &session)
	}

	session.Sessions = append(session.Sessions, PersistedMessage{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Round:     round,
		Type:      msgType,
		Messages:  *messages,
	})

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(PERSIST_FILE, data, 0644)
}
