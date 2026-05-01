package tools

import "time"

// ==================== task ====================

type TaskState string
const (
	TaskStatePending TaskState = "pending"
	TaskStateRunning TaskState = "running"
	TaskStateDone    TaskState = "done"
	TaskStateFailed   TaskState = "failed"
)

type Task struct {
	ID          int       `json:"id"`
	Subject     string    `json:"subject"`
	Description string    `json:"description"`
	State       TaskState `json:"state"`
	BlockedBy   []int     `json:"blocked_by"`
	Owner       string    `json:"owner"`

	// === background task fields ===
	IsBackground bool   `json:"is_background"`
	Command      string `json:"command,omitempty"`
	Output       string `json:"output,omitempty"`
}

type TaskUpdateOptions struct {
	Status          TaskState
	AddBlockedBy    []int
	RemoveBlockedBy []int
}

// ==================== worktree ====================
type WorktreeState string

const (
	WorktreeStateActive  WorktreeState = "active"
	WorktreeStateRemoved WorktreeState = "removed"
	WorktreeStateKept    WorktreeState = "kept"
)

type Worktree struct {
	TaskID    int           `json:"task_id,omitempty"`
	Name      string        `json:"name"`
	Path      string        `json:"path"`
	Branch    string        `json:"branch"`
	Status    WorktreeState `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

type EventType string
const (
	WorktreeCreateBefore EventType = "worktree.create.before"
	WorktreeCreateAfter  EventType = "worktree.create.after"
	WorktreeCreateFailed EventType = "worktree.create.failed"
	WorktreeRemoveBefore EventType = "worktree.remove.before"
	WorktreeRemoveAfter  EventType = "worktree.remove.after"
	WorktreeRemoveFail   EventType = "worktree.remove.fail"
	WorktreeKeepAlive    EventType = "worktree.keep"
	TaskCompleted        EventType = "task.completed"
)

