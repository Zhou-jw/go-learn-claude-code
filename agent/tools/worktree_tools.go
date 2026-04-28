package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

type Worktree struct {
	task_id int
	Name    string
	Path    string
	Branch  string
	status  WorktreeState
}

// WorktreeManager 管理所有 worktree
type WorktreeManager struct {
	rootdir    string
	index_path string
	worktrees  map[int]*Worktree
	mu         sync.RWMutex
	bus        EventBus // 内嵌 eventbus，不独立
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

func NewWorktreeManager(workdir string) *WorktreeManager {
	worktree_dir := filepath.Join(workdir, ".worktrees")
	index_path := filepath.Join(worktree_dir, "index.jsonl")
	return &WorktreeManager{
		rootdir:    workdir,
		index_path: index_path,
		worktrees:  make(map[int]*Worktree), // task_id as key
		mu:         sync.RWMutex{},
		bus:        NewEventBus(worktree_dir),
	}
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
		m.worktrees[wt.task_id] = wt
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

func (m *WorktreeManager) CreateWorktree(task *Task, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task_id := task.ID
	if _, exists := m.worktrees[task_id]; exists {
		return fmt.Errorf("worktree for task %d already exists", task_id)
	}

	wt := &Worktree{
		task_id: task_id,
		Name:    name,
		Path:    filepath.Join(m.rootdir, ".worktrees", name),
		Branch:  name,
		status:  WorktreeStateActive,
	}
	m.worktrees[task_id] = wt

	return m.save_index_map()
}
