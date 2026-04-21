package agent

import (
	"glcc/agent/team"
	"glcc/agent/tools"
	"os"
)

var WORKDIR string
var CHILD_TOOLS = SubagentTools()
var PARENT_TOOLS = MainAgentTools()

func InitAll() {
	WORKDIR, _ = os.Getwd()

	init_persist()
	tools.Init_tools(WORKDIR)
	init_prompt()
	team.Init_teammate_manager(WORKDIR)

}
