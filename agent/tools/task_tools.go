package tools

import (
	"encoding/json"
	"fmt"
	"glcc/console"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var stateMark = map[TaskState]string{
	TaskStatePending: "[ ]",
	TaskStateRunning: "[·]",
	TaskStateDone:    "[√]",
	TaskStateFailed:  "[✗]",
}

func (s TaskState) is_valid() bool {
	switch s {
	case TaskStatePending, TaskStateRunning, TaskStateDone, TaskStateFailed:
		return true
	default:
		return false
	}
}

type TaskManager struct {
	dir     string
	tasks   map[int]*Task
	nextid  int
	mu      sync.Mutex
	bg_chan chan *Task
	bashCfg *BashConfig
}

func newTaskManager(dir string, bashCfg *BashConfig) (*TaskManager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	tm := &TaskManager{
		dir:     dir,
		tasks:   make(map[int]*Task),
		bg_chan: make(chan *Task, 20),
		bashCfg: bashCfg,
	}
	tm.load_all_from_disk()
	tm.nextid = tm.max_id() + 1
	return tm, nil
}

func NewTaskManager(workdir string) *TaskManager {
	bashCfg := NewBashConfig(workdir)
	tm, err := newTaskManager(filepath.Join(workdir, ".task"), bashCfg)
	if err != nil {
		console.Red("Fail to init task manager: %v", err)
		return nil
	}
	return tm
}

func (m *TaskManager) load_all_from_disk() error {
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

func (m *TaskManager) next_id() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.nextid
	m.nextid++
	return id
}

func (m *TaskManager) max_id() int {
	max_id := 0
	for id := range m.tasks {
		if id > max_id {
			max_id = id
		}
	}
	return max_id
}

func (m *TaskManager) load(taskid int) (*Task, error) {
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

func (m *TaskManager) persist(task *Task) error {
	path := filepath.Join(m.dir, "task_"+strconv.Itoa(task.ID)+".json")
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *TaskManager) CreateTask(subject string, description string) (*Task, error) {
	id := m.next_id()

	task := &Task{
		ID:          id,
		Subject:     subject,
		Description: description,
		State:       TaskStatePending,
		BlockedBy:   []int{},
		Owner:       "",
	}
	if err := m.persist(task); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.tasks[task.ID] = task
	m.mu.Unlock()

	return task, nil
}

func (m *TaskManager) CreateBgTask(command string) (*Task, error) {
	id := m.next_id()

	task := &Task{
		ID:           id,
		Description:  fmt.Sprintf("Background Task: %s", command),
		State:        TaskStateRunning,
		BlockedBy:    []int{},
		IsBackground: true,
		Command:      command,
	}
	if err := m.persist(task); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.tasks[task.ID] = task
	m.mu.Unlock()

	return task, nil
}

func (m *TaskManager) RunBgTask(task *Task, timeout int64) {
	output, err := RunBashWithConfig(m.bashCfg, task.Command, timeout)

	m.mu.Lock()
	if err != nil {
		task.State = TaskStateFailed
	} else {
		task.State = TaskStateDone
	}
	task.Output = output
	m.mu.Unlock()

	_ = m.persist(task)
	m.bg_chan <- task
}

func (m *TaskManager) Drain() []*Task {
	var list []*Task
	for {
		select {
		case t := <-m.bg_chan:
			list = append(list, t)
		default:
			return list
		}
	}
}

func (m *TaskManager) Update(taskid int, status TaskState, add_Blocked_by []int, remove_Blocked_by []int) error {
	task, ok := m.tasks[taskid]
	if !ok {
		return fmt.Errorf("task %d not found", taskid)
	}

	if status.is_valid() {
		task.State = status
	}

	if status == TaskStateDone {
		m.clear_dependency(taskid)
	}

	newBlock := make(map[int]bool)
	for _, bid := range task.BlockedBy {
		newBlock[bid] = true
	}
	for _, bid := range add_Blocked_by {
		newBlock[bid] = true
	}
	for _, bid := range remove_Blocked_by {
		delete(newBlock, bid)
	}

	task.BlockedBy = make([]int, 0, len(newBlock))
	for bid := range newBlock {
		task.BlockedBy = append(task.BlockedBy, bid)
	}

	return m.persist(task)
}

func (m *TaskManager) clear_dependency(taskid int) {
	for _, task := range m.tasks {
		changed := false
		new_Blocked := make([]int, 0, len(task.BlockedBy))
		for _, id := range task.BlockedBy {
			if id != taskid {
				new_Blocked = append(new_Blocked, id)
			} else {
				changed = true
			}
		}
		if changed {
			task.BlockedBy = new_Blocked
			if err := m.persist(task); err != nil {
				return
			}
		}
	}
}

func (m *TaskManager) list_all() string {
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
		line := m.FormatTask(id)
		lines = append(lines, line)
	}
	return strings.Join(lines, "")
}

func (m *TaskManager) Get(taskID int) *Task {
	task, ok := m.tasks[taskID]
	if !ok {
		return nil
	}
	return task
}

func (m *TaskManager) FormatTask(taskID int) string {
	task := m.Get(taskID)
	if task == nil {
		return "Task not found."
	}

	mark, ok := stateMark[task.State]
	if !ok {
		mark = "[?]"
	}

	Blocked := ""
	if len(task.BlockedBy) > 0 {
		strs := make([]string, len(task.BlockedBy))
		for i, v := range task.BlockedBy {
			strs[i] = strconv.Itoa(v)
		}
		Blocked = fmt.Sprintf(" (Blocked by: %s)", strings.Join(strs, ", "))
	}

	return fmt.Sprintf("%s #%d: %s %s", mark, taskID, task.Subject, Blocked)
}

func (m *TaskManager) Exists(taskID int) bool {
	_, ok := m.tasks[taskID]
	return ok
}

func (m *TaskManager) BindWorktree(taskID int, worktreeName string) error {
	task, ok := m.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %d not found", taskID)
	}
	task.Worktree = worktreeName
	return m.persist(task)
}

func (m *TaskManager) UnbindWorktree(taskID int) error {
	task, ok := m.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %d not found", taskID)
	}
	task.Worktree = ""
	return m.persist(task)
}

func (m *TaskManager) ListAll() string {
	return m.list_all()
}
