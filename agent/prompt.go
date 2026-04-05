package agent

import (
	"fmt"
	"os"
	"sync"
)

// 单例初始化，保证安全
var (
	initOnce         sync.Once
	sysPrompt        string
	subagentSysPrompt string
	cwd              string
)

// 初始化一次：所有全局变量只在这里初始化
func initAll() {
	cwd, _ = os.Getwd()

	// 主提示词
	sysPrompt = fmt.Sprintf(`You are a coding agent at %s.
Use load_skill to access specialized knowledge before tackling unfamiliar topics.
Skills available:
%s`, cwd, SKILL_LOADER.GetDescriptions())

	// 子提示词
	subagentSysPrompt = fmt.Sprintf(
		"You are a coding subagent at %s. Use bash to solve tasks. Complete the given task, then summarize your findings.",
		cwd,
	)
}

// 获取主提示词（对外公开）
func Sys_prompt() string {
	initOnce.Do(initAll)
	return sysPrompt
}

// 获取子提示词（如需使用）
func Subagent_sys_prompt() string {
	initOnce.Do(initAll)
	return subagentSysPrompt
}