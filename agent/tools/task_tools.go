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

var TASKMGR *TaskManager

var stateMark = map[TaskState]string{
	TaskStatePending: "[ ]",
	TaskStateRunning: "[·]",
	TaskStateDone:    "[√]",
	TaskStateFailed:  "[✗]",
}

// 校验状态是否合法（替代原来的枚举检查）
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
}

func new_task_manager(dir string) (*TaskManager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	tm := &TaskManager{
		dir:     dir,
		tasks:   make(map[int]*Task),
		bg_chan: make(chan *Task, 20),
	}
	tm.load_all_from_disk()
	tm.nextid = tm.max_id() + 1
	return tm, nil
}

func NewTaskManager(workdir string) *TaskManager {
	tm, err := new_task_manager(filepath.Join(workdir, ".task"))
	if err != nil {
		console.Red("Fail to init task manager: %v", err)
		return nil
	}
	return tm
}

func init_task_manager(workdir string) {
	var err error
	TASKMGR, err = new_task_manager(filepath.Join(workdir, ".task"))
	if err != nil {
		console.Red("Fail to init task manager")
	}
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

func (m *TaskManager) run_bg_task(task *Task, timeout int64, workdir string) {
	output, err := RunBash(task.Command, timeout, workdir)

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

func DrainBackgroundTasks() []*Task {
	if deprecatedTools != nil && deprecatedTools.TaskManager != nil {
		return deprecatedTools.TaskManager.Drain()
	}
	if TASKMGR == nil {
		return nil
	}
	return TASKMGR.Drain()
}

func (m *TaskManager) Update(taskid int, status TaskState, add_blocked_by []int, remove_blocked_by []int) error {
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
	for _, bid := range add_blocked_by {
		newBlock[bid] = true
	}
	for _, bid := range remove_blocked_by {
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

// GetTaskByID —— 明确：根据ID获取任务实体（业务逻辑用）
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

	blocked := ""
	if len(task.BlockedBy) > 0 {
		strs := make([]string, len(task.BlockedBy))
		for i, v := range task.BlockedBy {
			strs[i] = strconv.Itoa(v)
		}
		blocked = fmt.Sprintf(" (blocked by: %s)", strings.Join(strs, ", "))
	}

	return fmt.Sprintf("%s #%d: %s %s", mark, taskID, task.Subject, blocked)
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
