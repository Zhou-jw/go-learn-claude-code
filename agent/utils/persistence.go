package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

type PersistedMessage struct {
	Timestamp string                   `json:"timestamp"`
	Round     int                      `json:"round"`
	Type      string                   `json:"type"`
	Messages  []anthropic.MessageParam `json:"messages"`
}

type PersistedSession struct {
	Sessions []PersistedMessage `json:"sessions"`
}

type Persister struct {
	persistDir  string
	persistFile string
	saveLock    *sync.Mutex
}

func NewPersister(baseDir string) *Persister {
	timestamp := time.Now().Format("20060102_150405")
	persistDir := fmt.Sprintf("%s/agent_logs/%s", baseDir, timestamp)
	persistFile := fmt.Sprintf("%s/messages.jsonl", persistDir)
	return &Persister{
		persistDir:  persistDir,
		persistFile: persistFile,
		saveLock:    new(sync.Mutex),
	}
}

func (p *Persister) Dir() string {
	return p.persistDir
}

func (p *Persister) ToolResultsDir() string {
	return fmt.Sprintf("%s/tool_results", p.persistDir)
}

func (p *Persister) TranscriptDir() string {
	return fmt.Sprintf("%s/transcripts", p.persistDir)
}

func (p *Persister) SaveMessages(messages *[]anthropic.MessageParam, agentName string, round int, msgType string) error {
	p.saveLock.Lock()
	defer p.saveLock.Unlock()

	if err := os.MkdirAll(p.persistDir, 0755); err != nil {
		return err
	}

	msg := PersistedMessage{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Round:     round,
		Type:      msgType,
		Messages:  *messages,
	}

	var filePath string
	if agentName == "" {
		filePath = p.persistFile
	} else {
		filePath = fmt.Sprintf("%s/%s_messages.jsonl", p.persistDir, agentName)
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(msg)
}

type PersisterFunc func(p *Persister) error

func Init_persist() *Persister {
	return NewPersister(".")
}
