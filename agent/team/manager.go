package team

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"

	"glcc/agent/team/bus"
)

var TEAMMATE_MGR *TeammateManager

func Init_teammate_manager(workdir string) {
	TEAMMATE_MGR = NewTeammateManager(workdir)
}

type TeammateManager struct {
	workdir string
	dir     string
	config  TeamConfig
	threads map[string]*memberThread
	mu      sync.RWMutex
	bus     bus.Bus
}

func NewTeammateManager(workdir string) *TeammateManager {
	teamDir := filepath.Join(workdir, ".team")
	os.MkdirAll(teamDir, 0755)

	mgr := &TeammateManager{
		workdir: workdir,
		dir:     teamDir,
		threads: make(map[string]*memberThread),
		bus:     bus.NewJSONLBus(workdir),
	}
	mgr.config = LoadTeamConfig(teamDir)
	return mgr
}

func (m *TeammateManager) Spawn(name, role, prompt string, client anthropic.Client, modelID string) string {
	member := FindMember(&m.config, name)
	if member != nil {
		if member.Status != "idle" && member.Status != "shutdown" {
			return fmt.Sprintf("Error: '%s' is currently %s", name, member.Status)
		}
		member.Status = "working"
		member.Role = role
	} else {
		member = &Member{Name: name, Role: role, Status: "working"}
		m.config.Members = append(m.config.Members, *member)
	}
	m.saveConfig()

	m.mu.Lock()
	m.threads[name] = &memberThread{
		name:    name,
		role:    role,
		prompt:  prompt,
		workdir: m.workdir,
		client:  client,
		modelID: modelID,
		mgr:     m,
	}
	m.mu.Unlock()

	go m.threads[name].run()
	return fmt.Sprintf("Spawned '%s' (role: %s)", name, role)
}

func (m *TeammateManager) ListAll() string {
	if len(m.config.Members) == 0 {
		return "No teammates."
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("Team: %s", m.config.TeamName))
	for _, member := range m.config.Members {
		lines = append(lines, fmt.Sprintf("  %s (%s): %s", member.Name, member.Role, member.Status))
	}
	return strings.Join(lines, "\n")
}

func (m *TeammateManager) MemberNames() []string {
	names := make([]string, len(m.config.Members))
	for i, member := range m.config.Members {
		names[i] = member.Name
	}
	return names
}

func (m *TeammateManager) Bus() bus.Bus {
	return m.bus
}

func (m *TeammateManager) saveConfig() {
	SaveTeamConfig(m.dir, m.config)
}
