package tools

import "github.com/anthropics/anthropic-sdk-go"

func CoreTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"command": map[string]any{"type": "string"},
				},
				Required: []string{"command"},
			},
			"bash",
		),
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"path": map[string]any{"type": "string"},
				},
				Required: []string{"path"},
			},
			"read_file",
		),
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"path":    map[string]any{"type": "string"},
					"content": map[string]any{"type": "string"},
				},
				Required: []string{"path", "content"},
			},
			"write_file",
		),
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"path":     map[string]any{"type": "string"},
					"old_text": map[string]any{"type": "string"},
					"new_text": map[string]any{"type": "string"},
				},
				Required: []string{"path", "old_text", "new_text"},
			},
			"edit_file",
		),
	}
}

func TaskTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
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
						"enum": []string{"pending", "in_progress", "done"},
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
		),
	}
}

func CommonTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
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
					"command": map[string]any{
						"type":        "string",
						"description": "The command to run in the background",
					},
					"timeout": map[string]any{
						"type":        "integer",
						"description": "The timeout for the background command",
					},
				},
			},
			"background_run", // 工具名称
		),
	}
}

func SpawnSubagentTool() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"prompt": map[string]any{"type": "string"},
				},
				Required: []string{"prompt"},
			},
			"task", // 工具名称
		),
	}
}

func TeamCommonTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{

		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"to":      map[string]any{"type": "string"},
					"content": map[string]any{"type": "string"},
					"msg_type": map[string]any{"type": "string",
						"enum": []string{"message", "broadcast", "shutdown_request", "shutdown_response", "plan_approval_response"},
					},
				},
				Required: []string{"to", "content"},
			},
			"send_message",
		),
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type:       "object",
				Properties: map[string]any{},
			},
			"read_inbox",
		),
	}
}

func LeadTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"name":   map[string]any{"type": "string"},
					"role":   map[string]any{"type": "string"},
					"prompt": map[string]any{"type": "string"},
				},
				Required: []string{"path", "old_text", "new_text"},
			},
			"spawn_teammate",
		),
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
			},
			"list_teammate",
		),
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"teammate": map[string]any{"type": "string"},
				},
				Required: []string{"teammate"},
			},
			"shutdown_request",
		),
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"request_id": map[string]any{"type": "string"},
				},
				Required: []string{"request_id"},
			},
			"shutdown_response",
		),
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"request_id": map[string]any{"type": "string"},
					"approve":    map[string]any{"type": "boolean"},
				},
				Required: []string{"request_id", "approve"},
			},
			"plan_approval",
		),
	}
}

func TeammateTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{

		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"request_id": map[string]any{"type": "string"},
					"approve":    map[string]any{"type": "boolean"},
				},
				Required: []string{"request_id", "approve"},
			},
			"shutdown_response",
		),
		anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"plan": map[string]any{"type": "string"},
				},
				Required: []string{"plan"},
			},
			"plan_approval",
		),
	}
}
