package tools

import "fmt"

// Task Handlers
func (r *ToolRegistry) handleTaskCreate(input map[string]any) string {
	subject, _ := input["subject"].(string)
	description, _ := input["description"].(string)
	
	task, err := r.tasks.CreateTask(subject, description)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Task created: %d", task.ID)
}

func (r *ToolRegistry) handleTaskUpdate(input map[string]any) string {
	taskID, ok := r.getInt(input, "task_id")
	if !ok {
		return "Error: task_id is required"
	}

	statusStr, ok := input["status"].(string)
	if !ok {
		return "Error: status is required"
	}
	status := TaskState(statusStr)

	if err := r.tasks.Update(taskID, status, r.getIntArray(input, "addBlockedBy"), r.getIntArray(input, "removeBlockedBy")); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Task %d updated", taskID)
}

func (r *ToolRegistry) handleTaskList(input map[string]any) string {
	return r.tasks.ListAll()
}

func (r *ToolRegistry) handleTaskGet(input map[string]any) string {
	taskID, ok := r.getInt(input, "task_id")
	if !ok {
		return "Error: task_id is required"
	}
	return r.tasks.FormatTask(taskID)
}

func (r *ToolRegistry) handleBackgroundRun(input map[string]any) string {
	cmd, ok := input["command"].(string)
	if !ok {
		return "Error: command is required"
	}

	timeout, ok := input["timeout"].(float64)
	if !ok {
		timeout = 120
	}

	task, err := r.tasks.CreateBgTask(cmd)
	if err != nil {
		return "Error: " + err.Error()
	}

	go r.runBgTask(task, int64(timeout), r.workdir)
	return fmt.Sprintf("Task created: %d\n", task.ID)
}

func (r *ToolRegistry) runBgTask(task *Task, timeout int64, workdir string) {
	output, err := RunBash(task.Command, timeout, workdir)

	if err != nil {
		task.State = TaskStateFailed
	} else {
		task.State = TaskStateDone
	}
	task.Output = output

	_ = r.tasks.Update(task.ID, task.State, nil, nil)
}

// 系统命令 Handler
func (r *ToolRegistry) handleBash(input map[string]any) string {
	cmd, _ := input["command"].(string)
	// RunBash 是之前定义的工具函数
	out, err := RunBash(cmd, 120, r.workdir)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return out
}

// Worktree Handlers
func (r *ToolRegistry) handleWorktreeCreate(input map[string]any) string {
	taskID, _ := r.getInt(input, "task_id")
	name, _ := input["name"].(string)
	
	if err := r.worktree.CreateWorktree(taskID, name, "HEAD"); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Worktree %s created for task %d", name, taskID)
}

func (r *ToolRegistry) handleWorktreeList(input map[string]any) string {
	return r.worktree.ListAll()
}

// 文件操作 Handlers
func (r *ToolRegistry) handleReadFile(input map[string]any) string {
	path, ok := input["path"].(string)
	if !ok {
		return "Error: path is required"
	}
	limit := 0
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}
	return RunRead(path, r.workdir, limit)
}

func (r *ToolRegistry) handleWriteFile(input map[string]any) string {
	path, ok := input["path"].(string)
	if !ok {
		return "Error: path is required"
	}
	content, ok := input["content"].(string)
	if !ok {
		return "Error: content is required"
	}
	return RunWrite(path, content, r.workdir)
}

func (r *ToolRegistry) handleEditFile(input map[string]any) string {
	path, ok := input["path"].(string)
	if !ok {
		return "Error: path is required"
	}
	oldText, ok := input["old_text"].(string)
	if !ok {
		return "Error: old_text is required"
	}
	newText, ok := input["new_text"].(string)
	if !ok {
		return "Error: new_text is required"
	}
	return RunEdit(path, oldText, newText, r.workdir)
}

// Todo Handler
func (r *ToolRegistry) handleTodo(input map[string]any) string {
	itemArray, ok := input["todo_items"].([]any)
	if !ok {
		return "Error: todo_items must be an array"
	}
	var todos []TodoItem
	for _, todo := range itemArray {
		todoMap, ok := todo.(map[string]any)
		if !ok {
			return "Error: invalid todo item format"
		}

		id, _ := todoMap["id"].(string)
		text, _ := todoMap["text"].(string)
		statusStr, _ := todoMap["status"].(string)
		status := TodoStatus(statusStr)

		todos = append(todos, TodoItem{
			ID:     id,
			Text:   text,
			Status: status,
		})
	}
	renderStr, err := r.todoMgr.Update(todos)
	if err != nil {
		return "Error: " + err.Error()
	}
	return renderStr
}

// Skill Handler
func (r *ToolRegistry) handleLoadSkill(input map[string]any) string {
	name, ok := input["name"].(string)
	if !ok {
		return "Error: name must be a string"
	}
	return r.skillLoader.GetContent(name)
}

// Compact Handler
func (r *ToolRegistry) handleCompact(input map[string]any) string {
	return "Manual compression requested."
}

// Task Bind Worktree Handler
func (r *ToolRegistry) handleTaskBindWorktree(input map[string]any) string {
	taskID, ok := r.getInt(input, "task_id")
	if !ok {
		return "Error: task_id is required"
	}
	worktree, ok := input["worktree"].(string)
	if !ok {
		return "Error: worktree is required"
	}
	if err := r.tasks.BindWorktree(taskID, worktree); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Task %d bound to worktree %s", taskID, worktree)
}