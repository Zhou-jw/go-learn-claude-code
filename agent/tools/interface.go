package tools

// ==================== 接口定义 ====================

type TaskProvider interface {
	Exists(taskID int) bool
	Get(taskID int) *Task
	Update(taskID int, opts TaskUpdateOptions) error
	BindWorktree(taskID int, worktreeName string) error
	UnbindWorktree(taskID int) error
}

type EventBus interface {
	Emit(event EventType, data any)
}