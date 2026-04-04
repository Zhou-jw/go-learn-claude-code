package agent

import (
	"fmt"
	"os"
)

var cwd, _ = os.Getwd()
var Sys_prompt = fmt.Sprintf("You are a coding agent at %s.Use the todo tool to plan multi-step tasks. Mark in_progress before starting, completed when done.Prefer tools over prose.", cwd)
var Subagent_sys_prompt = fmt.Sprintf("You are a coding subagent at %s. Use bash to solve tasks. Complete the given task, then summarize your findings.", cwd)