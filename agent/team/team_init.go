package team

import "github.com/anthropics/anthropic-sdk-go"

var TEAMMATE_MGR *TeammateManager
var client *anthropic.Client
var modelid string

func Init_teammate_manager(workdir string, c *anthropic.Client, modelID string) {
	TEAMMATE_MGR = NewTeammateManager(workdir)
	client = c
	modelid = modelID
}