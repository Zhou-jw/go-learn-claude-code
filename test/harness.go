package tester

import (
	"os"

	"glcc/agent"
	"glcc/agent/tools"
)

var testAgent *agent.Agent

func GetTestAgent() *agent.Agent {
	if testAgent == nil {
		cwd, _ := os.Getwd()
		testAgent = agent.NewAgent(cwd)
	}
	return testAgent
}

func GetTestTools() *tools.Tools {
	return GetTestAgent().Tools
}
