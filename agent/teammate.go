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

	team.TEAMMATE_MGR.Bus().Send("lead", to, content, msg_type, map[string]any{})
	return fmt.Sprintf("Message sent to %s: %s\n", to, content)
}