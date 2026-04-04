package agent

import (
	"fmt"
	"os"
)

var cwd, _ = os.Getwd()
var Sys_prompt = fmt.Sprintf("You are a coding agent at %s. Use bash to solve tasks. Act, don't explain.", cwd)
var Subagent_sys_prompt = fmt.Sprintf("You are a coding agent at %s. Use bash to solve tasks. Act, don't explain.", cwd)