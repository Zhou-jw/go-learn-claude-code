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

func Init() {
	WORKDIR, _ = os.Getwd()
	TODOMGR = NewTodoManager()
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
			ID: id, 
			Text: text,
			Status: status,
		})
	}
	render_str, err := TODOMGR.Update(todos)
	if err != nil {
		return "Error: " + err.Error()
	}
	return render_str
}

var TOOL_HANDLERS = map[string]ToolHandler{
	"bash":       handleBash,
	"read_file":  handleReadFile,
	"write_file": handleWriteFile,
	"edit_file":  handleEditFile,
	"todo":       handleTodo,
}

var TOOLS = []anthropic.ToolUnionParam{
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
}

func DispatchTool(name string, input map[string]any) string {
	handler, ok := TOOL_HANDLERS[name]
	if !ok {
		return fmt.Sprintf("Unknown tool: %s", name)
	}
	return handler(input)
}
