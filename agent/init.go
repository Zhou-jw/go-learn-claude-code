package agent

import (
	"glcc/agent/team"
	"os"
)

var WORKDIR string

func InitAll() {
	WORKDIR, _ = os.Getwd()

	init_prompt()
	init_todo_manager()
	Init_task_manager(WORKDIR)
	
	init_persist()
	team.Init_teammate_manager(WORKDIR)
	
}
