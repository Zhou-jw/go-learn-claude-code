package tester

import (
	toolsPkg "glcc/agent/tools"
	"strings"
	"testing"
	"time"
)

func TestHandleBackgroundRun(t *testing.T) {
	tools := GetTestTools()

	input := map[string]any{
		"command": "echo 'hello world'",
		"timeout": float64(30),
	}

	result := tools.Registry.Dispatch("background_run", input)
	t.Logf("Result: %s", result)

	if !strings.Contains(result, "Task created:") {
		t.Fatalf("Failed to create background task: %s", result)
	}

	time.Sleep(time.Second)

	tasks := tools.TaskManager.Drain()
	if len(tasks) == 0 {
		t.Fatalf("drain returned no tasks, expected to get completion signal")
	}

	found := false
	for _, task := range tasks {
		if task.State == toolsPkg.TaskStateDone || task.State == toolsPkg.TaskStateFailed {
			found = true
			t.Logf("Task %d completed with state: %s, output: %s", task.ID, task.State, task.Output)
		}
	}
	if !found {
		t.Fatalf("No completed task found in drain results")
	}

	t.Log("✅ handle_background_run with drain test passed!")
}

func TestHandleBackgroundRunMultiple(t *testing.T) {
	tools := GetTestTools()

	tools.Registry.Dispatch("background_run", map[string]any{
		"command": "echo 'task1'",
		"timeout": 30,
	})
	tools.Registry.Dispatch("background_run", map[string]any{
		"command": "echo 'task2'",
		"timeout": 30,
	})

	time.Sleep(500 * time.Millisecond)

	tasks := tools.TaskManager.Drain()
	if len(tasks) < 2 {
		t.Fatalf("Expected at least 2 tasks, got %d", len(tasks))
	}

	t.Logf("✅ drain retrieved %d completed tasks", len(tasks))
}
