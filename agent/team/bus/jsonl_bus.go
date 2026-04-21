package bus

import (
	"encoding/json"
	"fmt"
	"glcc/console"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type JSONLBus struct {
	dir string
}

func NewJSONLBus(workdir string) *JSONLBus {
	dir := filepath.Join(workdir, ".team", "inbox")
	os.MkdirAll(dir, 0755)
	return &JSONLBus{dir: dir}
}

func (b *JSONLBus) inboxPath(name string) string {
	return filepath.Join(b.dir, name+".jsonl")
}

func (b *JSONLBus) Send(from, to, content, msgType string, extra map[string]any) string {
	if !ValidMsgTypes[msgType] {
		return fmt.Sprintf("Error: Invalid type '%s'", msgType)
	}

	msg := Message{
		Type:      msgType,
		From:      from,
		To:        to,
		Content:   content,
		Timestamp: time.Now().Unix(),
		Extra:     extra,
	}

	path := b.inboxPath(to)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer f.Close()

	f.Write(append(data, '\n'))
	return fmt.Sprintf("Sent %s to %s", msgType, to)
}

func (b *JSONLBus) ReadInbox(name string) []Message {
	path := b.inboxPath(name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		console.Red("fail to read inbox %s.", path)
		return nil
	}

	var msgs []Message
	for line := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			msgs = append(msgs, msg)
		}
	}

	os.WriteFile(path, []byte{}, 0644)
	return msgs
}

func (b *JSONLBus) Broadcast(sender string, content string, members []string) string {
	count := 0
	for _, name := range members {
		if name != sender {
			b.Send(sender, name, content, "broadcast", nil)
			count++
		}
	}
	return fmt.Sprintf("Broadcast to %d teammates", count)
}
