package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"glcc/console"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	REPO_ROOT string
	WORKTREE  *WorktreeManager
)

func init_worktree(workdir string) {
	var err error
	REPO_ROOT, err = detect_repo_root(workdir)
	//(TODO) err may be caused by git not being installed or the directory not being a git repo
	if err != nil {
		REPO_ROOT = workdir
	}
}

// *********************  Event *******************

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

type EventBus interface {
	Emit(event EventType, data any)
}

func NewEventBus(dir string) EventBus {
	return NewJsonlBus(dir)
}

type JsonlBus struct {
	dir string
}

func NewJsonlBus(dir string) EventBus {
	return &JsonlBus{dir: dir}
}

func (b *JsonlBus) Emit(event EventType, data any) {

}

// *********************** Worktree *************************
type WorktreeState string

const (
	WorktreeStateActive  WorktreeState = "active"
	WorktreeStateRemoved WorktreeState = "removed"
	WorktreeStateKept    WorktreeState = "kept"
)

type TaskProvider interface {
	Exists(taskID int) bool
	BindWorktree(taskID int, worktreeName string) error
}

type Worktree struct {
	TaskID    int           `json:"task_id,omitempty"`
	Name      string        `json:"name"`
	Path      string        `json:"path"`
	Branch    string        `json:"branch"`
	Status    WorktreeState `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

// WorktreeManager 管理所有 worktree
type WorktreeManager struct {
	rootdir       string
	index_path    string
	worktrees     map[int]*Worktree
	mu            sync.RWMutex
	bus           EventBus // 内嵌 eventbus，不独立
	task_provider TaskProvider
}

func detect_repo_root(workdir string) (string, error) {
	timeout := 10
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = workdir

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("Error: Timeout (%ds)", timeout)
	}
	if err != nil {
		return "", err
	}

	repo_root := strings.TrimSpace(string(output))

	return repo_root, nil
}

func NewWorktreeManager(workdir string, task_provider TaskProvider) *WorktreeManager {
	worktree_dir := filepath.Join(workdir, ".worktrees")
	index_path := filepath.Join(worktree_dir, "index.jsonl")
	wt := &WorktreeManager{
		rootdir:       worktree_dir,
		index_path:    index_path,
		worktrees:     make(map[int]*Worktree), // task_id as key
		mu:            sync.RWMutex{},
		bus:           NewEventBus(worktree_dir),
		task_provider: task_provider,
	}

	wt.load_index_map()

	return wt
}

// deprecated: use load_index_map() instead
func (m *WorktreeManager) load_index_vector() error {
	data, err := os.ReadFile(m.index_path)
	if err != nil {
		return err
	}

	var worktrees []Worktree
	if err = json.Unmarshal(data, &worktrees); err != nil {
		return err
	}

	for i := range worktrees {
		// for _, wt := range worktrees -> all wt pointers are the same
		// m.worktrees[wt.Name] = &wt   -> wrong!
		wt := &worktrees[i]
		m.worktrees[wt.TaskID] = wt
	}
	return nil
}

func (m *WorktreeManager) load_index_map() error {
	data, err := os.ReadFile(m.index_path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &m.worktrees)
}

func (m *WorktreeManager) save_index_map() error {
	data, err := json.MarshalIndent(m.worktrees, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.index_path, data, 0644)
}

func (m *WorktreeManager) is_git_repo() (bool, error) {
	timeout := 10
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = m.rootdir

	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return false, fmt.Errorf("command timed out after %d seconds", timeout)
	}

	if err != nil {
		return false, err
	}

	out := strings.TrimSpace(string(output))
	return out == "true", nil
}

var validWorktreeNameRegex = regexp.MustCompile(`^[A-Za-z0-9._-]{1,40}$`)

func (m *WorktreeManager) is_name_valid(name string) (bool, error) {
	err_msg := "Invalid worktree name. Use 1-40 chars: letters, numbers, ., _, -"
	if name == "" {
		return false, errors.New(err_msg)
	}

	// 正则全匹配校验
	if !validWorktreeNameRegex.MatchString(name) {
		return false, errors.New(err_msg)
	}

	return true, nil
}

func (m *WorktreeManager) run_git(args []string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (m *WorktreeManager) get_worktree(name string) (*Worktree, bool ) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, wt_ptr := range m.worktrees {
		if wt_ptr.Name == name {
			return wt_ptr, true
		}
	}
	return nil, false
}

func (m *WorktreeManager) CreateWorktree(task_id int, name string, base_ref string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.worktrees[task_id]; exists {
		return fmt.Errorf("worktree for task %d already exists", task_id)
	}

	_, err := m.is_name_valid(name)
	if err != nil {
		return err
	}

	if base_ref == "" {
		base_ref = "HEAD"
	}

	path := filepath.Join(m.rootdir, name)
	branch := fmt.Sprintf("wt/%s", name)

	_, exist := m.worktrees[task_id]
	if exist {
		return fmt.Errorf("worktree for task %d already exists", task_id)
	}

	m.bus.Emit(WorktreeCreateBefore, map[string]any{
		"task":     map[string]any{"id": task_id},
		"worktree": map[string]any{"name": name, "base_ref": base_ref},
	})

	output, err := m.run_git([]string{"worktree", "add", "-b", branch, path, base_ref})
	if err != nil {
		m.bus.Emit(WorktreeCreateFailed, map[string]any{
			"task":     map[string]any{"id": task_id},
			"worktree": map[string]any{"name": name, "base_ref": base_ref},
			"error":    err.Error(),
		})
		return err
	}
	console.Info("%s", output)

	// persist index map
	err = m.save_index_map()
	if err != nil {
		return err
	}

	// create worktree entry after index map is persisted
	wt := &Worktree{
		TaskID:    task_id,
		Name:      name,
		Path:      path,
		Branch:    branch,
		Status:    WorktreeStateActive,
		CreatedAt: time.Now(),
	}
	m.worktrees[task_id] = wt

	// bind worktree
	m.task_provider.BindWorktree(task_id, name)

	m.bus.Emit(WorktreeCreateAfter, map[string]any{
		"task":     map[string]any{"id": task_id},
		"worktree": wt,
	})

	return nil
}

func (m *WorktreeManager) ListAll() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 没有 worktree 时返回提示（对应 Python if not wts: ...）
	if len(m.worktrees) == 0 {
		return "No worktrees in index."
	}

	var lines []string
	// 遍历所有 worktree（对应 Python for wt in wts）
	for _, wt := range m.worktrees {
		// 拼接 task_id 后缀（对应 Python suffix = ...）
		var suffix string
		if wt.TaskID != 0 {
			suffix = fmt.Sprintf(" task=%d", wt.TaskID)
		}

		branch := wt.Branch
		if branch == "" {
			branch = "-"
		}

		status := string(wt.Status)
		if status == "" {
			status = "unknown"
		}

		line := fmt.Sprintf("[%s] %s -> %s (%s)%s",
			status,
			wt.Name,
			wt.Path,
			branch,
			suffix,
		)
		lines = append(lines, line)
	}

	// 用换行连接返回（对应 Python "\n".join(lines)）
	return strings.Join(lines, "\n")
}

func (m *WorktreeManager) Status(name string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	wt, ok := m.get_worktree(name)
	if !ok {
		return fmt.Sprintf("Error: Unknown worktree '%s'", name)
	}

	path := wt.Path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Sprintf("Error: Worktree path missing: %s", path)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "git", "status", "--short", "--branch")
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err.Error()
	}
	
	text := strings.TrimSpace(string(output))
	if text =="" {
		return "Clean Worktree.\n"
	}
	return text
}