package agent

import "os"

var WORKDIR string

func InitAll() {
	WORKDIR, _ = os.Getwd()

	init_prompt()
	init_tool_use()
	init_persist()
}
