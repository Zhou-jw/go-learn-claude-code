package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// ==================== TaskState ====================
type TaskState string

const (
	TaskStatePending TaskState = "pending"
	TaskStateRunning TaskState = "running"
	TaskStateDone    TaskState = "done"
)

var stateMark = map[TaskState]string{
	TaskStatePending: "[ ]",
	TaskStateRunning: "[>]",
	TaskStateDone:    "[√]",
}

// 校验状态是否合法（替代原来的枚举检查）
func (s TaskState) IsValid() bool {
	switch s {
	case TaskStatePending, TaskStateRunning, TaskStateDone:
		return true
	default:
		return false
	}
}

type Task struct {
	ID          int       `json:"id"`
	Subject     string    `json:"subject"`
	Description string    `json:"description"`
	State       TaskState `json:"state"`
	BlockedBy   []int     `json:"blocked_by"`
	Owner       string    `json:"owner"`
}

type TaskMgr struct {
	dir    string
	tasks  map[int]*Task
	nextid int
	mu     sync.Mutex
}

func NewTaskMgr(dir string) (*TaskMgr, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	tm := &TaskMgr{
		dir:   dir,
		tasks: make(map[int]*Task),
	}
	tm.load_all_from_disk()
	tm.nextid = tm.max_id() + 1
	return tm, nil
}

func (m *TaskMgr) load_all_from_disk() error {
	files, err := filepath.Glob(filepath.Join(m.dir, "task_*.json"))
	if err != nil {
		return err
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}
		m.tasks[task.ID] = &task
	}
	return nil
}

func (m *TaskMgr) next_id() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nextid
}

func (m *TaskMgr) max_id() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(m.dir, "task_*.json"))
	if err != nil || len(files) == 0 {
		return 0
	}

	max_id := 0
	for _, f := range files {
		base := filepath.Base(f)
		parts := strings.Split(base, "_")
		if len(parts) < 2 {
			continue
		}
		idstr := strings.TrimSuffix(parts[1], ".json")
		id, err := strconv.Atoi(idstr)
		if err != nil {
			continue
		}
		if id > max_id {
			max_id = id
		}
	}
	return max_id
}

func (m *TaskMgr) load(taskid int) (*Task, error) {
	path := filepath.Join(m.dir, "task_"+strconv.Itoa(taskid)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (m *TaskMgr) persist(task *Task) error {
	path := filepath.Join(m.dir, "task_"+strconv.Itoa(task.ID)+".json")
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *TaskMgr) NewTask(subject string, description string) (*Task, error) {
	task := &Task{
		ID:          m.next_id(),
		Subject:     subject,
		Description: description,
		State:       TaskStatePending,
		BlockedBy:   []int{},
		Owner:       "",
	}
	if err := m.persist(task); err != nil {
		return nil, err
	}

	m.tasks[task.ID] = task
	m.nextid++

	return task, nil
}

func (m *TaskMgr) Update(taskid int, status TaskState, add_blocked_by []int, remove_blocked_by []int) error {
	task, ok := m.tasks[taskid]
	if !ok {
		return fmt.Errorf("task %d not found", taskid)
	}

	if status.IsValid() {
		task.State = status
	}

	if status == TaskStateDone {
		m.clear_dependency(taskid)
	}

	add_set := make(map[int]bool)
	if len(add_blocked_by) > 0 {
		for _, id := range task.BlockedBy {
			add_set[id] = true
		}
		for _, id := range add_blocked_by {
			add_set[id] = true
		}
	}

	rm_set := make(map[int]bool)
	if len(remove_blocked_by) > 0 {
		for _, id := range remove_blocked_by {
			rm_set[id] = true
		}
	}

	new_blocked := make([]int, 0, len(add_set))
	for id := range add_set {
		if !rm_set[id] {
			new_blocked = append(new_blocked, id)
		}
	}
	task.BlockedBy = new_blocked

	if err := m.persist(task); err != nil {
		return err
	}

	return nil
}

func (m *TaskMgr) clear_dependency(taskid int) {
	for _, task := range m.tasks {
		changed := false
		new_blocked := make([]int, 0, len(task.BlockedBy))
		for _, id := range task.BlockedBy {
			if id != taskid {
				new_blocked = append(new_blocked, id)
			} else {
				changed = true
			}
		}
		if changed {
			task.BlockedBy = new_blocked
			if err := m.persist(task); err != nil {
				return
			}
		}
	}
}

func (m *TaskMgr) ListAll() string {
	if len(m.tasks) == 0 {
		return "No tasks."
	}

	ids := make([]int, 0, len(m.tasks))
	for id := range m.tasks {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	var lines []string
	for _, id := range ids {
		line := m.Get(id)
		lines = append(lines, line)
	}
	return strings.Join(lines, "")
}

func (m *TaskMgr) Get(taskid int) string {
	task, ok := m.tasks[taskid]
	if !ok {
		return "Task not found."
	}
	mark, ok := stateMark[task.State]
	if !ok {
		mark = "[?]"
	}

	blocked := ""
	if len(task.BlockedBy) > 0 {
		strs := make([]string, len(task.BlockedBy))
		for i, v := range task.BlockedBy {
			strs[i] = strconv.Itoa(v)
		}
		blocked = fmt.Sprintf(" (blocked by: %s)", strings.Join(strs, ", "))
	}
	line := fmt.Sprintf("%s #%d: %s %s\n", mark, taskid, task.Subject, blocked)
	return line
}
