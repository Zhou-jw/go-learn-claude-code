package tester

import (
	"glcc/agent"
	"os"
	"strings"
	"testing"
)

// 初始化一次，避免重复创建
func TestMain(m *testing.M) {
	agent.Init()
	
	exitCode := m.Run()

	os.Exit(exitCode)
}

var name = "todo"
// 测试 todo 工具：正常添加任务
func TestTodoTool_Works(t *testing.T) {
	// 这就是 AI 传给你的 input 格式
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

	// 调用工具
	result := agent.DispatchTool(name,input)
	t.Log(result)

	// 验证没有报错
	if strings.Contains(result, "Error") {
		t.Fatalf("todo 工具失败: %s", result)
	}

	t.Log("✅ todo 功能正常！")
}

// 测试：不能同时有2个 in_progress
func TestTodoTool_NoTwoInProgress(t *testing.T) {
	input := map[string]any{
		"todo_items": []any{
			map[string]any{"id": "1", "text": "任务1", "status": "in_progress"},
			map[string]any{"id": "2", "text": "任务2", "status": "in_progress"},
		},
	}

	result := agent.DispatchTool(name, input)
	if !strings.Contains(result, "only one task can be in_progress") {
		t.Fatalf("应该阻止两个进行中任务，但没有: %s", result)
	}
	t.Log("✅ 阻止双进行中任务正常！")
}

// 测试：状态必须是合法值
func TestTodoTool_InvalidStatus(t *testing.T) {
	input := map[string]any{
		"todo_items": []any{
			map[string]any{"id": "1", "text": "测试", "status": "invalid"},
		},
	}

	result := agent.DispatchTool(name, input)
	if !strings.Contains(result, "invalid status") {
		t.Fatalf("应该拒绝无效状态，但没有: %s", result)
	}
	t.Log("✅ 无效状态拦截正常！")
}

// 测试：text 不能为空
func TestTodoTool_EmptyText(t *testing.T) {
	input := map[string]any{
		"todo_items": []any{
			map[string]any{"id": "1", "text": "", "status": "pending"},
		},
	}

	result := agent.DispatchTool(name, input)
	if !strings.Contains(result, "text required") {
		t.Fatalf("应该拦截空文本，但没有: %s", result)
	}
	t.Log("✅ 空文本拦截正常！")
}
