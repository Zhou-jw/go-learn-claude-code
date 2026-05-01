package tools

import (
	"fmt"
	"path/filepath"
)

type ToolRegistry struct {
	tasks         TaskProvider
	worktree      *WorktreeManager
	workdir       string
	todoMgr       *TodoManager
	skillLoader   *SkillLoader
	dangerousCmds []string
	handlers      map[string]ToolHandler
}

func NewToolRegistry(tp TaskProvider, wm *WorktreeManager, workdir string, tm *TodoManager, sl *SkillLoader) *ToolRegistry {
	cfg := NewBashConfig(workdir)
	reg := &ToolRegistry{
		tasks:         tp,
		worktree:      wm,
		workdir:       workdir,
		todoMgr:       tm,
		skillLoader:   sl,
		dangerousCmds: cfg.DangerousCommands,
		handlers:      make(map[string]ToolHandler),
	}
	reg.initDefaultHandlers()
	return reg
}

func (r *ToolRegistry) initDefaultHandlers() {
	r.handlers = map[string]ToolHandler{
		"task_create":        r.handleTaskCreate,
		"task_update":        r.handleTaskUpdate,
		"task_list":          r.handleTaskList,
		"task_get":           r.handleTaskGet,
		"background_run":     r.handleBackgroundRun,
		"task_bind_worktree": r.handleTaskBindWorktree,
		"bash":               r.handleBash,
		"read_file":          r.handleReadFile,
		"write_file":         r.handleWriteFile,
		"edit_file":          r.handleEditFile,
		"todo":               r.handleTodo,
		"load_skill":         r.handleLoadSkill,
		"compact":            r.handleCompact,
		"worktree_create":    r.handleWorktreeCreate,
		"worktree_list":      r.handleWorktreeList,
	}
}

func (r *ToolRegistry) RegisterHandler(name string, handler ToolHandler) {
	r.handlers[name] = handler
}

type Tools struct {
	TaskManager     *TaskManager
	WorktreeManager *WorktreeManager
	TodoManager     *TodoManager
	SkillLoader     *SkillLoader
	Workdir         string
	Registry        *ToolRegistry
}

func NewTools(workdir string) *Tools {
	tm := NewTaskManager(workdir)
	wm := NewWorktreeManager(workdir, tm)
	todoMgr := NewTodoManager()
	sl := NewSkillLoader(filepath.Join(workdir, "skills"))

	reg := NewToolRegistry(tm, wm, workdir, todoMgr, sl)

	return &Tools{
		TaskManager:     tm,
		WorktreeManager: wm,
		TodoManager:     todoMgr,
		SkillLoader:     sl,
		Workdir:         workdir,
		Registry:        reg,
	}
}

// Dispatch 根据工具名分发到具体的方法
func (r *ToolRegistry) Dispatch(name string, input map[string]any) string {
	handlers := r.getHandlers()
	handler, ok := handlers[name]
	if !ok {
		return fmt.Sprintf("Unknown tool: %s", name)
	}
	return handler(input)
}

func (r *ToolRegistry) getHandlers() map[string]ToolHandler {
	return r.handlers
}

// --- 通用类型转换辅助工具 ---

func (r *ToolRegistry) getInt(m map[string]any, key string) (int, bool) {
	val, ok := m[key].(float64)
	return int(val), ok
}

func (r *ToolRegistry) getIntArray(m map[string]any, key string) []int {
	var res []int
	if raw, ok := m[key].([]any); ok {
		for _, v := range raw {
			if f, ok := v.(float64); ok {
				res = append(res, int(f))
			}
		}
	}
	return res
}
