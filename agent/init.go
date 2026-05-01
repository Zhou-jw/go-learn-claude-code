package agent

import (
	"glcc/agent/team"
	"glcc/agent/tools"
	"glcc/agent/utils"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
)

type Agent struct {
	Workdir string
	Tools   *tools.Tools
	Persist *utils.Persister
	Compact *CompactConfig
}

func NewAgent(workdir string) *Agent {
	persister := utils.NewPersister(workdir)
	compactCfg := InitCompactConfig(persister, workdir)
	t := tools.NewTools(workdir)
	RegisterTeammateTools(t)
	initPromptWithTools(t)
	return &Agent{
		Workdir: workdir,
		Tools:   t,
		Persist: persister,
		Compact: compactCfg,
	}
}

func CreateAgent(client *anthropic.Client, modelID string) *Agent{
	workdir, _ := os.Getwd()
	ag := NewAgent(workdir)
	team.Init_teammate_manager(workdir, client, modelID, ag.Persist)
	return ag
}

func InitForTest() {
	workdir, _ := os.Getwd()
	NewAgent(workdir)
}

func SetTools(t *tools.Tools) {
}
