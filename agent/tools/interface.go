package tools

// ==================== 接口定义 ====================

type TaskProvider interface {
	Exists(taskID int) bool
	Get(taskID int) *Task
	FormatTask(taskID int) string
	ListAll() string
	CreateTask(subject string, description string) (*Task, error)
	CreateBgTask(command string) (*Task, error)
	Update(taskID int, status TaskState, addBlockedBy []int, removeBlockedBy []int) error
	BindWorktree(taskID int, worktreeName string) error
	UnbindWorktree(taskID int) error
}

type EventBus interface {
	Emit(event EventType, data any)
}

type ToolHandler func(input map[string]any) string