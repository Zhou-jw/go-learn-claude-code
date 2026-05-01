package team

import (
	"glcc/agent/utils"

	"github.com/anthropics/anthropic-sdk-go"
)

var TEAMMATE_MGR *TeammateManager

func Init_teammate_manager(workdir string, c *anthropic.Client, modelID string, persister *utils.Persister) {
	TEAMMATE_MGR = NewTeammateManager(workdir, persister, c, modelID)
}
