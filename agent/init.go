package agent

import (
	"glcc/agent/team"
	"glcc/agent/tools"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
)

var WORKDIR string
var CHILD_TOOLS = SubagentTools()
var PARENT_TOOLS = MainAgentTools()

func InitAll(client *anthropic.Client, modelID string) {
	WORKDIR, _ = os.Getwd()

	init_persist()
	tools.Init_tools(WORKDIR)
	init_prompt()
	team.Init_teammate_manager(WORKDIR, client, modelID)
}

func InitForTest() {
	WORKDIR, _ = os.Getwd()
	init_persist()
	tools.Init_tools(WORKDIR)
	init_prompt()
}
