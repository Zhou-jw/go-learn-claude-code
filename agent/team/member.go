package team

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"glcc/agent/tools"
	"glcc/agent/utils"
	"glcc/console"
)

type memberThread struct {
	name, role, prompt, workdir string
	client                      *anthropic.Client
	modelID                     string
	mgr                         *TeammateManager
}

func (mt *memberThread) run() {
	sysPrompt := fmt.Sprintf(`You are '%s', role: %s, at %s.
		Submit plans via plan_approval before major work.
		Respond to shutdown_request with shutdown_response.
		`, mt.name, mt.role, mt.mgr.dir)

	var messages []anthropic.MessageParam
	messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(mt.prompt)))

	teammate_tools := mt.get_tools()
	should_shutdown := false

	console.Green("[%s] started", mt.name)
	for round := range 50 {
		inbox := mt.mgr.bus.ReadInbox(mt.name)
		for _, msg := range inbox {
			msgJSON, _ := json.Marshal(msg)
			messages = append(messages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(string(msgJSON)),
			))
		}

		if len(inbox) == 0 && len(messages) >= 1 {
			console.Debug("messages' len: %v", len(messages))
			time.Sleep(300 * time.Millisecond)
			continue
		}

		console.Debug("[%s] call llm ", mt.name)
		resp, err := mt.client.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(mt.modelID),
			MaxTokens: 8000,
			System:    []anthropic.TextBlockParam{{Text: sysPrompt}},
			Messages:  messages,
			Tools:     teammate_tools,
		})
		if err != nil {
			break
		}
		console.Debug("[%s] AI 调用返回，err=%v", mt.name, err)
		utils.SaveMessages(&messages, mt.name, round, "teammate")

		// Append assistant message
		var assistantContent []anthropic.ContentBlockParamUnion
		var toolResults []anthropic.ContentBlockParamUnion

		for _, block := range resp.Content {
			switch b := block.AsAny().(type) {
			case anthropic.TextBlock:
				assistantContent = append(assistantContent, anthropic.NewTextBlock(b.Text))
			case anthropic.ToolUseBlock:
				inputJSON, _ := json.Marshal(b.Input)
				assistantContent = append(assistantContent, anthropic.NewToolUseBlock(b.ID, json.RawMessage(inputJSON), b.Name))

				// Execute tools
				output := mt.execTool(b.Name, b.Input, &should_shutdown)
				console.Info("  [%s] %s: %s\n", mt.name, b.Name, output[:min(120, len(output))])
				toolResults = append(toolResults, anthropic.NewToolResultBlock(b.ID, output, false))
			}
		}
		messages = append(messages, anthropic.NewAssistantMessage(assistantContent...))
		
		utils.SaveMessages(&messages, mt.name, round, "teammate")
		
		messages = append(messages, anthropic.NewUserMessage(toolResults...))


		if resp.StopReason != anthropic.StopReasonToolUse {
			console.Red("[%s] break: %s", mt.name, resp.StopReason)
			break
		}

		console.Debug("[%s] step over stop reason, should_shutdown is %v", mt.name, should_shutdown)

		if should_shutdown {
			shutdown_member(&mt.mgr.config, mt.name)
		}
		utils.SaveMessages(&messages, mt.name, round, "teammate")

	}

	// Set member to idle on exit
	if member := FindMember(&mt.mgr.config, mt.name); member != nil {
		if member.Status != "shutdown" {
			member.Status = "idle"
		}
		mt.mgr.saveConfig()
	}

	console.Red("teammate %s break: \n", mt.name)

	utils.SaveMessages(&messages, mt.name, 0, "teammate")
}

func (mt *memberThread) execTool(toolName string, input json.RawMessage, should_shutdown *bool) string {
	var args map[string]any
	json.Unmarshal(input, &args)

	switch toolName {
	case "bash":
		cmd, _ := args["command"].(string)
		result, err := tools.RunBash(cmd, 120, mt.workdir)
		if err != nil {
			return err.Error()
		}
		return result
	case "read_file":
		path, _ := args["path"].(string)
		limit := 0
		if l, ok := args["limit"].(float64); ok {
			limit = int(l)
		}
		return tools.RunRead(path, mt.workdir, limit)
	case "write_file":
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)
		return tools.RunWrite(path, content, mt.workdir)
	case "edit_file":
		path, _ := args["path"].(string)
		oldText, _ := args["old_text"].(string)
		newText, _ := args["new_text"].(string)
		return tools.RunEdit(path, oldText, newText, mt.workdir)
	case "send_message":
		to, _ := args["to"].(string)
		content, _ := args["content"].(string)
		msgType, _ := args["msg_type"].(string)
		if msgType == "" {
			msgType = "message"
		}
		return mt.mgr.Send(mt.name, to, content, msgType, nil)
	case "read_inbox":
		msgs := mt.mgr.bus.ReadInbox(mt.name)
		data, _ := json.MarshalIndent(msgs, "", "  ")
		return string(data)
	case "shutdown_response":
		req_id, _ := args["request_id"].(string)
		approve, _ := args["approve"].(bool)
		console.Debug("req_id is %s, approve is %v", req_id, approve)

		ok := mt.mgr.UpdateRequest(req_id, approve)
		console.Debug("update request is %v", ok)
		
		res := mt.mgr.Send(mt.name, "lead", "", "shutdown_response", nil)
		console.Debug("%s", res)
		
		if approve {
			*should_shutdown = true
		}
		return fmt.Sprintf("Shutdown response sent: %s, approve: %v", req_id, approve)
	}

	return fmt.Sprintf("Unknown tool: %s", toolName)
}

func (mt *memberThread) get_tools() []anthropic.ToolUnionParam {
	core_tools := tools.CoreTools()
	teammate_tools := tools.TeamCommonTools()
	teammate_tools = append(teammate_tools, tools.TeammateTools()...)
	return append(core_tools, teammate_tools...)
}
