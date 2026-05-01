package tester

import (
	"os"
	"path/filepath"
	"testing"

	"glcc/agent/team"
	"glcc/agent/team/bus"
	"glcc/agent/utils"

	"github.com/stretchr/testify/assert"
)

func setupTestTeamDir(t *testing.T) string {
	tmpDir := t.TempDir()
	teamDir := filepath.Join(tmpDir, ".team")
	os.MkdirAll(teamDir, 0755)
	return tmpDir
}

func GetTestPersister(dir string) *utils.Persister {
	return utils.NewPersister(dir)
}

func TestLoadTeamConfig(t *testing.T) {
	tmpDir := setupTestTeamDir(t)
	teamDir := filepath.Join(tmpDir, ".team")

	cfg := team.LoadTeamConfig(teamDir)

	assert.Equal(t, "default", cfg.TeamName)
	assert.Empty(t, cfg.Members)
}

func TestSaveAndLoadTeamConfig(t *testing.T) {
	tmpDir := setupTestTeamDir(t)
	teamDir := filepath.Join(tmpDir, ".team")

	cfg := team.TeamConfig{
		TeamName: "test-team",
		Members: []team.Member{
			{Name: "alice", Role: "coder", Status: "idle"},
			{Name: "bob", Role: "reviewer", Status: "working"},
		},
	}

	team.SaveTeamConfig(teamDir, cfg)

	loaded := team.LoadTeamConfig(teamDir)
	assert.Equal(t, "test-team", loaded.TeamName)
	assert.Equal(t, 2, len(loaded.Members))
}

func TestFindMember(t *testing.T) {
	cfg := team.TeamConfig{
		TeamName: "test-team",
		Members: []team.Member{
			{Name: "alice", Role: "coder", Status: "idle"},
		},
	}

	member := team.FindMember(&cfg, "alice")
	assert.Equal(t, "alice", member.Name)

	notFound := team.FindMember(&cfg, "bob")
	assert.Nil(t, notFound)
}

func TestNewTeammateManager(t *testing.T) {
	tmpDir := setupTestTeamDir(t)
	persister := utils.NewPersister(tmpDir)

	mgr := team.NewTeammateManager(tmpDir, persister, nil, "")

	assert.NotNil(t, mgr)
	assert.Equal(t, 0, len(mgr.MemberNames()))
}

func TestTeammateManager_ListAll(t *testing.T) {
	tmpDir := setupTestTeamDir(t)
	persister := utils.NewPersister(tmpDir)

	mgr := team.NewTeammateManager(tmpDir, persister, nil, "")

	result := mgr.ListAll()
	assert.Contains(t, result, "No teammates")
}

func TestJSONLBus_SendAndRead(t *testing.T) {
	tmpDir := setupTestTeamDir(t)
	b := bus.NewJSONLBus(tmpDir)

	result := b.Send("alice", "bob", "Hello Bob", "message", nil)
	assert.Contains(t, result, "Sent message to bob")

	msgs := b.ReadInbox("bob")
	assert.Len(t, msgs, 1)
	assert.Equal(t, "alice", msgs[0].From)
	assert.Equal(t, "bob", msgs[0].To)
	assert.Equal(t, "Hello Bob", msgs[0].Content)
	assert.Equal(t, "message", msgs[0].Type)
}

func TestJSONLBus_Drain(t *testing.T) {
	tmpDir := setupTestTeamDir(t)
	b := bus.NewJSONLBus(tmpDir)

	b.Send("alice", "bob", "msg1", "message", nil)
	b.Send("alice", "bob", "msg2", "message", nil)

	msgs := b.ReadInbox("bob")
	assert.Len(t, msgs, 2)

	msgs = b.ReadInbox("bob")
	assert.Len(t, msgs, 0)
}
