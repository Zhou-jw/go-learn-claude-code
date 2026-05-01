package tester

import (
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	_ = GetTestAgent()
}

func TestTodoTool_Works(t *testing.T) {
	input := map[string]any{
		"todo_items": []any{
			map[string]any{
				"id":     "1",
				"text":   "写Go代码",
				"status": "pending",
			},
			map[string]any{
				"id":     "2",
				"text":   "测试工具",
				"status": "in_progress",
			},
		},
	}

	result := GetTestTools().Registry.Dispatch("todo", input)
	t.Log(result)

	if strings.Contains(result, "Error") {
		t.Fatalf("todo 工具失败: %s", result)
	}

	t.Log("✅ todo 功能正常！")
}

func TestTodoTool_NoTwoInProgress(t *testing.T) {
	input := map[string]any{
		"todo_items": []any{
			map[string]any{"id": "1", "text": "任务1", "status": "in_progress"},
			map[string]any{"id": "2", "text": "任务2", "status": "in_progress"},
		},
	}

	result := GetTestTools().Registry.Dispatch("todo", input)
	if !strings.Contains(result, "only one task can be in_progress") {
		t.Fatalf("应该阻止两个进行中任务，但没有: %s", result)
	}
	t.Log("✅ 阻止双进行中任务正常！")
}

func TestTodoTool_InvalidStatus(t *testing.T) {
	input := map[string]any{
		"todo_items": []any{
			map[string]any{"id": "1", "text": "测试", "status": "invalid"},
		},
	}

	result := GetTestTools().Registry.Dispatch("todo", input)
	if !strings.Contains(result, "invalid status") {
		t.Fatalf("应该拒绝无效状态，但没有: %s", result)
	}
	t.Log("✅ 无效状态拦截正常！")
}

func TestTodoTool_EmptyText(t *testing.T) {
	input := map[string]any{
		"todo_items": []any{
			map[string]any{"id": "1", "text": "", "status": "pending"},
		},
	}

	result := GetTestTools().Registry.Dispatch("todo", input)
	if !strings.Contains(result, "text required") {
		t.Fatalf("应该拦截空文本，但没有: %s", result)
	}
	t.Log("✅ 空文本拦截正常！")
}
