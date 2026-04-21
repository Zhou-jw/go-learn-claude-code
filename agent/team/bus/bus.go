package bus

type Message struct {
	Type      string         `json:"type"`
	From      string         `json:"from"`
	To        string         `json:"to"`
	Content   string         `json:"content"`
	Timestamp int64          `json:"timestamp"`
	Extra     map[string]any `json:"extra,omitempty"`
}

var ValidMsgTypes = map[string]bool{
	"message":                true,
	"broadcast":              true,
	"shutdown_request":       true,
	"shutdown_response":      true,
	"plan_approval_response": true,
}

// Bus interface - 支持多种实现
type Bus interface {
	// Send a message to a specific teammate
	Send(from, to, content, msgType string, extra map[string]any) string

	// Read and drain inbox for a teammate
	ReadInbox(name string) []Message

	// Broadcast to all teammates except sender
	Broadcast(sender string, content string, members []string) string
}

// NewBus creates a Bus instance (根据配置选择实现)
func NewBus(impl string, workdir string) Bus {
	switch impl {
	case "jsonl":
		return NewJSONLBus(workdir)
	default:
		return NewJSONLBus(workdir)
	}
}
