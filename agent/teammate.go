package agent

import (
	"fmt"
	"glcc/agent/team"
	"glcc/agent/tools"
)

func init() {
	tools.RegisterHandler("spawn_teammate", handle_spawn_teammate)
	tools.RegisterHandler("list_teammate", handle_list_teammates)
	tools.RegisterHandler("send_message", handle_send_message)
	tools.RegisterHandler("shutdown_request", handle_shutdown_request)
	tools.RegisterHandler("shutdown_response", check_shutdown_status)
	tools.RegisterHandler("plan_approval", handle_plan_review)
}

func handle_spawn_teammate(input map[string]any) string {
	name, ok := input["name"].(string)
	if !ok {
		return "Error: name must be a string"
	}

	role, ok := input["role"].(string)
	if !ok {
		return "Error: role must be a string"
	}

	prompt, ok := input["prompt"].(string)
	if !ok {
		return "Error: prompt must be a string"
	}

	team.TEAMMATE_MGR.Spawn(name, role, prompt, "")
	return fmt.Sprintf("Teammate spawned: %s\n", name)
}

func handle_list_teammates(input map[string]any) string {
	return team.TEAMMATE_MGR.ListAll()
}

func handle_send_message(input map[string]any) string {
	to, ok := input["to"].(string)
	if !ok {
		return "Error: to must be a string"
	}

	content, ok := input["content"].(string)
	if !ok {
		return "Error: message must be a string"
	}

	msg_type, ok := input["msg_type"].(string)
	if !ok {
		return "Error: msg_type must be a string"
	}

	team.TEAMMATE_MGR.Send("lead", to, content, msg_type, map[string]any{})
	return fmt.Sprintf("Message sent to %s: %s\n", to, content)
}

func handle_shutdown_request(input map[string]any) string {
	teammate, ok := input["teammate"].(string)
	if !ok {
		return "Error: teammate must be a string"
	}

	content := "Please shut down gracefully."
	msg_type := "shutdown_request"
	req := team.TEAMMATE_MGR.NewRequest(msg_type, "lead", teammate)
	team.TEAMMATE_MGR.Send("lead", teammate, content, "shutdown_request", map[string]any{
		"request_id": req.ID,
	})
	return fmt.Sprintf("Shutdown request %s %s\n", req.ID, teammate)
}

func check_shutdown_status(input map[string]any) string {
	request_id, ok := input["request_id"].(string)
	if !ok {
		return "Error: request_id must be a string"
	}

	req := team.TEAMMATE_MGR.GetRequest(request_id)
	if req == nil {
		return fmt.Sprintf("No request found with id %s\n", request_id)
	}

	return fmt.Sprintf("Shutdown request %s %s\n", req.ID, req.Status)
}

func handle_plan_review(input map[string]any) string {
	request_id, ok := input["request_id"].(string)
	if !ok {
		return "Error: request_id must be a string"
	}

	approval, ok := input["approval"].(bool)
	if !ok {
		return "Error: approval must be a boolean"
	}

	feedback, ok := input["feedback"].(string)
	if !ok {
		feedback = ""
	}

	req := team.TEAMMATE_MGR.GetRequest(request_id)
	if req == nil {
		return fmt.Sprintf("No request found with id %s\n", request_id)
	}

	if approval {
		req.Status = "approved"
	} else {
		req.Status = "rejected"
	}

	msg_type := "plan_approval_response"

	team.TEAMMATE_MGR.Send("lead", req.From, feedback, msg_type, map[string]any{})

	return ""
}
