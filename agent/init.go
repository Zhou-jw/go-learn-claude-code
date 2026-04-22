package agent

import (
	"glcc/agent/team"
	"glcc/agent/tools"
	"glcc/agent/utils"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
)

var WORKDIR string
var CHILD_TOOLS = SubagentTools()
var PARENT_TOOLS = MainAgentTools()

func InitAll(client *anthropic.Client, modelID string) {
	WORKDIR, _ = os.Getwd()

	utils.Init_persist()
	init_context_compact()
	tools.Init_tools(WORKDIR)
	init_prompt()
	team.Init_teammate_manager(WORKDIR, client, modelID)
}

func InitForTest() {
	WORKDIR, _ = os.Getwd()
	utils.Init_persist()
	init_context_compact()
	tools.Init_tools(WORKDIR)
	init_prompt()
}
