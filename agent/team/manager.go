package team

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"glcc/agent/team/bus"
	"glcc/agent/utils"
	"glcc/console"

	"github.com/anthropics/anthropic-sdk-go"
)

type TeammateManager struct {
	workdir   string
	dir       string
	config    TeamConfig
	threads   map[string]*memberThread
	mu        sync.RWMutex
	bus       bus.Bus
	protocol  *Protocol
	persister *utils.Persister
	client    *anthropic.Client
	modelID   string
}

func NewTeammateManager(workdir string, persister *utils.Persister, client *anthropic.Client, modelID string) *TeammateManager {
	teamDir := filepath.Join(workdir, ".team")
	os.MkdirAll(teamDir, 0755)

	mgr := &TeammateManager{
		workdir:   workdir,
		dir:       teamDir,
		threads:   make(map[string]*memberThread),
		bus:       bus.NewJSONLBus(workdir),
		protocol:  NewProtocol(),
		persister: persister,
		client:    client,
		modelID:   modelID,
	}
	mgr.config = LoadTeamConfig(teamDir)
	return mgr
}

func (m *TeammateManager) Spawn(name, role, prompt string, modelID string) string {
	if modelID == "" {
		modelID = m.modelID
	}
	member := FindMember(&m.config, name)
	if member != nil {
		if member.Status != "idle" && member.Status != "shutdown" {
			return fmt.Sprintf("Error: '%s' is currently %s \n", name, member.Status)
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
		name:      name,
		role:      role,
		prompt:    prompt,
		workdir:   m.workdir,
		client:    m.client,
		modelID:   modelID,
		mgr:       m,
		persister: m.persister,
	}
	m.mu.Unlock()

	console.Debug("Spawned '%s' (role: %s)\n", name, role)
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
	return strings.Join(lines, "\n") + "\n"
}

func (m *TeammateManager) MemberNames() []string {
	names := make([]string, len(m.config.Members))
	for i, member := range m.config.Members {
		names[i] = member.Name
	}
	return names
}

func (m *TeammateManager) ReadInbox(from string) []bus.Message {
	return m.bus.ReadInbox(from)
}

func (m *TeammateManager) Send(from, to, content, msg_type string, data map[string]any) string {
	return m.bus.Send(from, to, content, msg_type, data)
}

func (m *TeammateManager) NewRequest(reqType, from, to string) *Request {
	return m.protocol.NewRequest(reqType, from, to)
}

func (m *TeammateManager) GetRequest(request_id string) *Request {
	req, ok := m.protocol.GetRequest(request_id)
	if !ok {
		return nil
	}
	return req
}

func (m *TeammateManager) UpdateRequest(request_id string, approve bool) bool {
	return m.protocol.Update(request_id, approve)
}

func (m *TeammateManager) saveConfig() {
	SaveTeamConfig(m.dir, m.config)
}
