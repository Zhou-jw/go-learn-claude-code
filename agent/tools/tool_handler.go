package tools

import (
	"fmt"
)

var dangerousCommands = []string{
	"rm -rf /",
	"sudo",
	"shutdown",
	"reboot",
	"> /dev/",
}

type ToolHandler func(input map[string]any) string

func handleBash(input map[string]any) string {
	command, ok := input["command"].(string)
	if !ok {
		return "Error: command is required"
	}
	out, err := RunBash(command, 120, WORKDIR)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return out
}

func handleReadFile(input map[string]any) string {
	path, ok := input["path"].(string)
	if !ok {
		return "Error: path is required"
	}
	limit := 0
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}
	return RunRead(path, WORKDIR, limit)
}

func handleWriteFile(input map[string]any) string {
	path, ok := input["path"].(string)
	if !ok {
		return "Error: path is required"
	}
	content, ok := input["content"].(string)
	if !ok {
		return "Error: content is required"
	}
	return RunWrite(path, content, WORKDIR)
}

func handleEditFile(input map[string]any) string {
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
	return RunEdit(path, oldText, newText, WORKDIR)
}

func handleTodo(input map[string]any) string {
	item_array, ok := input["todo_items"].([]any)
	if !ok {
		return "Error: todo_items must be an array"
	}
	var todos []TodoItem
	for _, todo := range item_array {
		todo_map, ok := todo.(map[string]any)
		if !ok {
			return "Error: invalid todo item format"
		}

		id, _ := todo_map["id"].(string)
		text, _ := todo_map["text"].(string)
		status_str, _ := todo_map["status"].(string)
		status := TodoStatus(status_str)

		todos = append(todos, TodoItem{
			ID:     id,
			Text:   text,
			Status: status,
		})
	}
	render_str, err := TODOMGR.Update(todos)
	if err != nil {
		return "Error: " + err.Error()
	}
	return render_str
}

func handle_load_skill(input map[string]any) string {
	name, ok := input["name"].(string)
	if !ok {
		return "Error: name must be a string"
	}
	return SKILL_LOADER.GetContent(name)
}

func handle_compact(input map[string]any) string {
	return "Manual compression requested."
}

func handle_background_run(input map[string]any) string {
	cmd, ok := input["command"].(string)
	if !ok {
		return "Error: cmd must be a string"
	}

	timeout, ok := input["timeout"].(float64)
	if !ok {
		timeout = 120
	}

	task, err := TASKMGR.create_bg_task(cmd)
	if err != nil {
		return "Error: " + err.Error()
	}

	go TASKMGR.run_bg_task(task, int64(timeout), WORKDIR)
	return fmt.Sprintf("Task created: %d\n", task.ID)
}

func handle_task_create(input map[string]any) string {
	subject, ok := input["subject"].(string)
	if !ok {
		return "Error: subject must be a string"
	}
	description, ok := input["description"].(string)
	if !ok {
		return "Error: description must be a string"
	}
	task, err := TASKMGR.create_task(subject, description)
	if err != nil {
		return "Error: " + err.Error()
	}
	return fmt.Sprintf("Task created: %d\n", task.ID)
}

func handle_task_update(input map[string]any) string {
	task_id_f64, ok := input["task_id"].(float64)
	if !ok {
		return "Error: task_id must be a number"
	}
	task_id := int(task_id_f64)

	status_str, ok := input["status"].(string)
	if !ok {
		return "Error: status must be a string"
	}
	status := TaskState(status_str)

	var add_blocked_by []int
	if raw, ok := input["addBlockedBy"]; ok && raw != nil {
		if arr, ok := raw.([]any); ok {
			for _, v := range arr {
				if f, ok := v.(float64); ok {
					add_blocked_by = append(add_blocked_by, int(f))
				}
			}
		}
	}

	var remove_blocked_by []int
	if raw, ok := input["removeBlockedBy"]; ok && raw != nil {
		if arr, ok := raw.([]any); ok {
			for _, v := range arr {
				if f, ok := v.(float64); ok {
					remove_blocked_by = append(remove_blocked_by, int(f))
				}
			}
		}
	}

	err := TASKMGR.Update(task_id, status, add_blocked_by, remove_blocked_by)
	if err != nil {
		return fmt.Sprintf("Error: %s\n", err.Error())
	}
	return fmt.Sprintf("Task updated: %d\n", task_id)
}

func handle_task_list(input map[string]any) string {
	return TASKMGR.list_all()
}

func handle_task_get(input map[string]any) string {
	task_id_f64, ok := input["task_id"].(float64)
	if !ok {
		return "Error: task_id must be a string"
	}
	task_id := int(task_id_f64)
	return TASKMGR.Get(task_id)
}

var TOOL_HANDLERS = map[string]ToolHandler{
	"bash":           handleBash,
	"read_file":      handleReadFile,
	"write_file":     handleWriteFile,
	"edit_file":      handleEditFile,
	"todo":           handleTodo,
	"load_skill":     handle_load_skill,
	"compact":        handle_compact,
	"background_run": handle_background_run,
	"task_create":    handle_task_create,
	"task_update":    handle_task_update,
	"task_list":      handle_task_list,
	"task_get":       handle_task_get,
}

func DispatchTool(name string, input map[string]any) string {
	handler, ok := TOOL_HANDLERS[name]
	if !ok {
		return fmt.Sprintf("Unknown tool: %s", name)
	}
	return handler(input)
}

func RegisterHandler(name string, handler ToolHandler) {
	TOOL_HANDLERS[name] = handler
}
