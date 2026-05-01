package agent

import (
	"fmt"
	"glcc/agent/tools"
)

type PromptConfig struct {
	SystemPrompt      string
	SubagentSysPrompt string
	Workdir           string
}

var (
	sysPrompt         string
	subagentSysPrompt string
)

func initPromptWithTools(t *tools.Tools) {
	sysPrompt = fmt.Sprintf(`You are a coding agent team lead at %s. Use tools to solve tasks.
		Spawn teammates and communicate via inboxes.
		Prefer task_create/task_update/task_list for multi-step work. Use TodoWrite for short checklists.
		Use task for subagent delegation. Use load_skill for specialized knowledge.
		Use background_run for long-running commands.
		Use load_skill to access specialized knowledge before tackling unfamiliar topics.
		Skills available:
		%s`, t.Workdir, t.SkillLoader.GetDescriptions(),
	)

	subagentSysPrompt = fmt.Sprintf(
		"You are a coding subagent at %s. Use bash to solve tasks. Complete the given task, then summarize your findings.",
		t.Workdir,
	)
}

func Sys_prompt() string {
	return sysPrompt
}

func Subagent_sys_prompt() string {
	return subagentSysPrompt
}
