package agent

import (
	"errors"
	"fmt"
	"strings"
)

type TodoStatus string

const (
	Pending    TodoStatus = "pending"
	InProgress TodoStatus = "in_progress"
	Completed  TodoStatus = "completed"
)

func isValidStatus(status TodoStatus) bool {
	switch status {
	case Pending, InProgress, Completed:
		return true
	default:
		return false
	}
}

type TodoItem struct {
	ID     string     `json:"id"`
	Text   string     `json:"text"`
	Status TodoStatus `json:"status"`
}

type TodoManager struct {
	todos []TodoItem
}

func NewTodoManager() *TodoManager {
	return &TodoManager{
		todos: make([]TodoItem, 0),
	}
}

func (m *TodoManager) Update(todos []TodoItem) (string, error) {
	if len(todos) > 20 {
		return "", errors.New("Max 20 todos allowed")
	}

	validated := make([]TodoItem, 0, len(todos))
	in_progress_cnt := 0
	for idx, todo := range todos {
		text := todo.Text
		status := todo.Status
		id := todo.ID

		if id == "" {
			id = fmt.Sprintf("%d", idx+1)
		}
		// 文本非空校验
		if text == "" {
			return "", fmt.Errorf("item %s: text required", id)
		}
		// 枚举合法性校验（只能是 pending/in_progress/completed）
		if !isValidStatus(status) {
			return "", fmt.Errorf("item %s: invalid status '%s'", id, status)
		}

		if status == InProgress {
			in_progress_cnt++
		}

		validated = append(validated, TodoItem{
			ID:     id,
			Text:   text,
			Status: status,
		})
	}
	if in_progress_cnt > 1 {
		return "", errors.New("only one task can be in_progress at a time")
	}
	m.todos = validated
	return m.Render(), nil
}

func (m *TodoManager) Render() string {
	if len(m.todos) == 0 {
		return "No todos"
	}
	var lines []string
	completed_cnt := 0
	
	for _, todo := range m.todos {
		var marker string
		switch todo.Status {
		case Pending:
			marker = "[ ]"
		case InProgress:
			marker = "[>]"
		case Completed:
			marker = "[✓]"
			completed_cnt++
		}
		lines = append(lines, fmt.Sprintf("%s #%s: %s", marker, todo.ID , todo.Text))
	}
	return strings.Join(lines, "\n")
}
