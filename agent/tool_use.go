package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

var dangerousCommands = []string{
	"rm -rf /",
	"sudo",
	"shutdown",
	"reboot",
	"> /dev/",
}

var WORKDIR string
var TODOMGR *TodoManager
var TASKMGR *TaskMgr

func init() {
	WORKDIR, _ = os.Getwd()
	TODOMGR = NewTodoManager()
	TASKMGR, _ = NewTaskMgr(filepath.Join(WORKDIR, "tasks"))
}

func safePath(p string) (string, error) {
	absPath := filepath.Join(WORKDIR, p)
	absPath = filepath.Clean(absPath)
	workDirAbs, _ := filepath.Abs(WORKDIR)
	if !strings.HasPrefix(absPath, workDirAbs) {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}
	return absPath, nil
}

func runBash(command string) string {
	for _, d := range dangerousCommands {
		if strings.Contains(command, d) {
			return "Error: Dangerous command blocked"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = WORKDIR

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "Error: Timeout (120s)"
	}
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	out := strings.TrimSpace(string(output))
	if out == "" {
		return "(no output)"
	}
	if len(out) > 50000 {
		return out[:50000]
	}
	return out
}

func runRead(path string, limit int) string {
	safePath, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	text := string(content)
	if len(text) > 50000 {
		text = text[:50000]
	}

	lines := strings.Split(text, "\n")
	if limit > 0 && limit < len(lines) {
		lines = append(lines[:limit], fmt.Sprintf("... (%d more lines)", len(lines)-limit))
	}

	return strings.Join(lines, "\n")
}

func runWrite(path string, content string) string {
	safePath, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if err := os.WriteFile(safePath, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return fmt.Sprintf("Wrote %d bytes to %s", len(content), path)
}

func runEdit(path string, oldText string, newText string) string {
	safePath, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, oldText) {
		return fmt.Sprintf("Error: Text not found in %s", path)
	}

	newContent := strings.Replace(text, oldText, newText, 1)
	if err := os.WriteFile(safePath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return fmt.Sprintf("Edited %s", path)
}

type ToolHandler func(input map[string]any) string

func handleBash(input map[string]any) string {
	command, ok := input["command"].(string)
	if !ok {
		return "Error: command is required"
	}
	return runBash(command)
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
	return runRead(path, limit)
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
	return runWrite(path, content)
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
	return runEdit(path, oldText, newText)
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

func handle_task_create(input map[string]any) string {
	subject, ok := input["subject"].(string)
	if !ok {
		return "Error: subject must be a string"
	}
	description, ok := input["description"].(string)
	if !ok {
		return "Error: description must be a string"
	}
	task, err := TASKMGR.NewTask(subject, description)
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
	return TASKMGR.ListAll()
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
	"bash":        handleBash,
	"read_file":   handleReadFile,
	"write_file":  handleWriteFile,
	"edit_file":   handleEditFile,
	"todo":        handleTodo,
	"load_skill":  handle_load_skill,
	"compact":     handle_compact,
	"task_create": handle_task_create,
	"task_update": handle_task_update,
	"task_list":   handle_task_list,
	"task_get":    handle_task_get,
}

var CHILD_TOOLS = []anthropic.ToolUnionParam{
	anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The bash command to execute",
				},
			},
			Required: []string{"command"},
		},
		"bash",
	),
	anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to read",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of lines to read",
				},
			},
			Required: []string{"path"},
		},
		"read_file",
	),
	anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to write",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the file",
				},
			},
			Required: []string{"path", "content"},
		},
		"write_file",
	),
	anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to edit",
				},
				"old_text": map[string]any{
					"type":        "string",
					"description": "The exact text to replace",
				},
				"new_text": map[string]any{
					"type":        "string",
					"description": "The replacement text",
				},
			},
			Required: []string{"path", "old_text", "new_text"},
		},
		"edit_file",
	),
	anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type: "object",
			// Description: "Update task list. Track progress on multi-step tasks.",
			Properties: map[string]any{
				"todo_items": map[string]any{
					"type":        "array",
					"description": "List of todo items to update",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id": map[string]any{
								"type":        "string",
								"description": "Unique ID of the todo item",
							},
							"text": map[string]any{
								"type":        "string",
								"description": "Todo item text content",
							},
							"status": map[string]any{
								"type":        "string",
								"description": "Status of the todo item",
								"enum":        []string{"pending", "in_progress", "completed"},
							},
						},
						"required": []string{"id", "text", "status"},
					},
				},
			},
			Required: []string{"todo_items"},
		},
		"todo", // 工具名称
	),
	anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Name of the skill to load",
				},
			},
			Required: []string{"name"},
		},
		"load_skill", // 工具名称
	),
	anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]any{
				"focus": map[string]any{
					"type":        "string",
					"description": "What to preserve in the summary",
				},
			},
		},
		"compact", // 工具名称
	),
	anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]any{
				"subject": map[string]any{
					"type": "string",
				},
				"description": map[string]any{
					"type": "string",
				},
			},
			Required: []string{""},
		},
		"task_create",
	),
	anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]any{
				"task_id": map[string]any{
					"type": "integer",
				},
				"status": map[string]any{
					"type": "string",
				},
				"addBlockedBy": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "integer",
					},
				},
				"removeBlockedBy": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "integer",
					},
				},
			},
			Required: []string{"task_id", "status"},
		},
		"task_update",
	),
	anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type:       "object",
			Properties: map[string]any{},
			Required:   []string{},
		},
		"task_list",
	),
	anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]any{
				"task_id": map[string]any{
					"type": "integer",
				},
			},
			Required: []string{"task_id"},
		},
		"task_get",
	)}

var TASK_TOOL = anthropic.ToolUnionParamOfTool(
	anthropic.ToolInputSchemaParam{
		Type: "object",
		Properties: map[string]any{
			"prompt": map[string]any{
				"type": "string",
			},
		},
		Required: []string{"prompt"},
	},
	"task", // 工具名称
)

var PARENT_TOOLS = append(CHILD_TOOLS, TASK_TOOL)

func DispatchTool(name string, input map[string]any) string {
	handler, ok := TOOL_HANDLERS[name]
	if !ok {
		return fmt.Sprintf("Unknown tool: %s", name)
	}
	return handler(input)
}
