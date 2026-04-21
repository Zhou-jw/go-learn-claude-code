# Agent 模块设计文档

## 1. 模块划分

```
agent/
├── tools/           # 基础工具实现 (可被所有模块复用)
│   ├── bash.go     # RunBash
│   └── file.go     # RunRead, RunWrite, RunEdit
├── skill.go        # SkillLoader (s05)
├── task.go         # TaskManager (s07)
├── background.go   # BackgroundManager (s08)
├── tool_use.go    # 工具分发 + Schema定义 (组装所有工具)
├── bus/            # 消息总线接口 (可替换实现)
│   ├── bus.go      # Bus interface
│   └── jsonl_bus.go # JSONL实现
├── team/           # s09/s11 团队域
│   ├── config.go   # TeamConfig
│   ├── member.go   # Teammate loop (使用 tools/)
│   └── manager.go  # TeammateManager
├── worktree/       # s12 Worktree域
│   ├── manager.go  # WorktreeManager
│   └── events.go  # EventBus
└── agent_loop.go   # 主循环
```

**说明**：
- `tools/` 包提供基础工具实现，不含任何 agent 特定逻辑
- `tool_use.go` 只负责"组装"工具（Handler + Schema）
- team/member 可以直接调用 `tools.RunBash()` 等
- 不需要独立的 TodoManager - TaskManager 已足够
- Bus 抽象为 interface，支持 JSONL/Redis/Mem 等实现
- EventBus 放在 worktree/ 下

---

## 2. 依赖关系

```
                    ┌─────────────┐
                    │ tool_use.go │  (底层)
                    └──────┬──────┘
                           │
     ┌──────────────────────┼──────────────────────┐
     │                      │                      │
┌────▼────┐          ┌────▼────┐          ┌───▼────┐
│skill.go │          │task.go  │          │background│
└─────────┘          └────┬────┘          └────────┘
                           │
                    ┌──────▼──────┐
                    │worktree/    │
                    │manager.go │
                    └───────────┘
     ┌──────────────────────┼──────────────────────┐
     │                      │                      │
┌────▼────┐          ┌────▼────┐          ┌───▼────┐
│bus/     │          │team/member           │team/   │
│jsonl    │          └─────────┘          │manager │
└─────────┘                │             └────────┘
              ┌────────────┴────────────┐
              │         bus/bus.go       │
              └─────────────────────────┘
                                   │
                            ┌──────▼────��─┐
                            │ agent_loop  │  (组合所有)
                            └────────────┘
```

---

## 3. 模块详细设计

### 3.1 tool_use.go (共享底层)

职责：文件操作、命令执行

状态：无持久化

```go
// 公开函数
func RunBash(command string, timeout int64) (string, error)
func RunRead(path string, limit int) string
func RunWrite(path string, content string) string
func RunEdit(path string, oldText, newText string) string

// Tool handlers
var TOOL_HANDLERS = map[string]ToolHandler{
    "bash":        handleBash,
    "read_file":   handleReadFile,
    "write_file": handleWriteFile,
    "edit_file":  handleEditFile,
}
```

### 3.2 task.go (s07)

职责：持久化任务管理

状态：`.tasks/task_N.json`

```go
type Task struct {
    ID          int       `json:"id"`
    Subject     string    `json:"subject"`
    Description string    `json:"description"`
    Status      string    `json:"status"` // pending, in_progress, completed
    Owner       string    `json:"owner"`
    Worktree   string    `json:"worktree"` // 绑定 worktree 名
    BlockedBy  []int     `json:"blocked_by"`
    CreatedAt  float64   `json:"created_at"`
    UpdatedAt  float64   `json:"updated_at"`
}

type TaskManager struct {
    dir     string
    tasks  map[int]*Task
    nextid  int
    mu     sync.Mutex
}

func NewTaskManager(dir string) *TaskManager
func (m *TaskManager) Create(subject, description string) (*Task, error)
func (m *TaskManager) Get(id int) (*Task, error)
func (m *TaskManager) Update(id int, status string, owner string) error
func (m *TaskManager) BindWorktree(id int, worktree string, owner string) error
func (m *TaskManager) UnbindWorktree(id int) error
func (m *TaskManager) ListAll() string
```

### 3.3 background.go (s08)

职责：后台任务执行 + 通知

状态：内存 + TaskManager 持久化

```go
type BackgroundManager struct {
    tasks        map[string]*BgTask
    notifications chan BgNotification
    taskMgr      *TaskManager
}

type BgTask struct {
    ID       string
    Command  string
    Status   string // running, completed, error
    Result   string
}

func NewBackgroundManager(taskMgr *TaskManager) *BackgroundManager
func (m *BackgroundManager) Run(command string, timeout int64) string
func (m *BackgroundManager) Check(taskID string) string
func (m *BackgroundManager) Drain() []BgNotification
```

### 3.4 skill.go (s05)

职责：技能加载

状态：`.skills/*/SKILL.md`

```go
type SkillLoader struct {
    skills map[string]Skill
}

type Skill struct {
    Meta    map[string]string
    Body    string
}

func NewSkillLoader(dir string) *SkillLoader
func (l *SkillLoader) Load(name string) string
func (l *SkillLoader) Descriptions() string
```

---

### 3.5 bus/bus.go - Bus Interface

职责：消息传递抽象

```go
type Message struct {
    Type      string         `json:"type"`
    From     string         `json:"from"`
    To       string         `json:"to"`
    Content  string         `json:"content"`
    Timestamp float64       `json:"timestamp"`
    Extra    map[string]any `json:"extra,omitempty"`
}

var ValidMsgTypes = map[string]bool{
    "message": true,
    "broadcast": true,
    "shutdown_request": true,
    "shutdown_response": true,
    "plan_approval_response": true,
}

// Bus interface - 支持多种实现
type Bus interface {
    // Send a message to a specific teammate
    Send(from, to, content, msgType string, extra map[string]any) string
    
    // Read and drain inbox for a teammate
    ReadInbox(name string) []Message
    
    // Broadcast to all teammates except sender
    Broadcast(sender string, content string, members []string) string
}

// NewBus creates a Bus instance (根据配置选择实现)
func NewBus(impl string, workdir string) Bus
```

### 3.6 bus/jsonl_bus.go - JSONL 实现

职责：JSONL 文件存储

状态：`.team/inbox/*.jsonl`

```go
type JSONLBus struct {
    dir string
}

func NewJSONLBus(workdir string) *JSONLBus

func (b *JSONLBus) Send(from, to, content, msgType string, extra map[string]any) string {
    if !ValidMsgTypes[msgType] {
        return fmt.Sprintf("Error: Invalid type '%s'", msgType)
    }
    
    msg := Message{
        Type:      msgType,
        From:     from,
        To:       to,
        Content:  content,
        Timestamp: time.Now().Unix(),
        Extra:    extra,
    }
    
    path := filepath.Join(b.dir, to + ".jsonl")
    f, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE, 0644)
    json.NewEncoder(f).Encode(msg)
    f.Close()
    
    return fmt.Sprintf("Sent %s to %s", msgType, to)
}

func (b *JSONLBus) ReadInbox(name string) []Message {
    path := filepath.Join(b.dir, name + ".jsonl")
    data, _ := os.ReadFile(path)
    
    var msgs []Message
    for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
        var msg Message
        if json.Unmarshal([]byte(line), &msg) == nil {
            msgs = append(msgs, msg)
        }
    }
    
    os.WriteFile(path, []byte{}, 0644)
    return msgs
}

func (b *JSONLBus) Broadcast(sender, content string, members []string) string {
    count := 0
    for _, name := range members {
        if name != sender {
            b.Send(sender, name, content, "broadcast", nil)
            count++
        }
    }
    return fmt.Sprintf("Broadcast to %d teammates", count)
}
```

### 3.7 team/config.go (s09)

职责：团队配置持久化

状态：`.team/config.json`

```go
type Member struct {
    Name   string `json:"name"`
    Role  string `json:"role"`
    Status string `json:"status"` // idle, working, shutdown
}

type TeamConfig struct {
    TeamName string   `json:"team_name"`
    Members []Member `json:"members"`
}
```

### 3.7 team/manager.go (s09/s11)

职责：Teammate 生命周期管理

状态：依赖 bus/bus.go 和 team/config.go

```go
type TeammateManager struct {
    dir      string
    config   TeamConfig
    threads  map[string]*memberThread
    mu       sync.RWMutex
    bus      bus.Bus  // 接口
    taskMgr  *TaskManager
}

func NewTeammateManager(workdir string, taskMgr *TaskManager) (*TeammateManager, bus.Bus)
func (m *TeammateManager) Spawn(name, role, prompt string) string
func (m *TeammateManager) ListAll() string
func (m *TeammateManager) MemberNames() []string
```

### 3.8 team/member.go (s09/s11)

职责：Teammate 线程循环

状态：无持久化（依赖 manager 管理）

```go
type memberThread struct {
    name, role, prompt string
    done           chan struct{}
    client         anthropic.Client
    modelID        string
    mgr            *TeammateManager
}

func (mt *memberThread) run() {
    // Work phase: 50 rounds
    // Idle phase: poll for messages + auto-claim tasks
}

// tools for teammate
var MEMBER_TOOLS = []anthropic.ToolUnionParam{
    // base: bash, read_file, write_file, edit_file
    // team: send_message, idle, claim_task
}
```

### 3.9 worktree/events.go (s12)

职责：Worktree/Task 生命周期事件

状态：`.worktrees/events.jsonl`

```go
type EventBus struct {
    path string
}

func NewEventBus(path string) *EventBus
func (e *EventBus) Emit(event string, task, worktree map[string]any, err string)
func (e *EventBus) ListRecent(limit int) string
```

### 3.10 worktree/manager.go (s12)

职责：Git worktree 管理

状态：`.worktrees/index.json`

```go
type WorktreeEntry struct {
    Name      string  `json:"name"`
    Path      string  `json:"path"`
    Branch    string  `json:"branch"`
    TaskID    int     `json:"task_id"`
    Status    string  `json:"status"` // active, kept, removed
    CreatedAt float64 `json:"created_at"`
}

type WorktreeManager struct {
    repoRoot  string
    dir       string
    tasks     *TaskManager
    events    *EventBus
    gitAvailable bool
}

func NewWorktreeManager(repoRoot string, tasks *TaskManager) *WorktreeManager
func (m *WorktreeManager) Create(name string, taskID int, baseRef string) string
func (m *WorktreeManager) ListAll() string
func (m *WorktreeManager) Status(name string) string
func (m *WorktreeManager) Run(name, command string) string
func (m *WorktreeManager) Keep(name string) string
func (m *WorktreeManager) Remove(name string, force, completeTask bool) string
```

---

## 4. 工具分配

| 工具 | 模块 | 对应 Python |
|------|------|-------------|
| bash | tool_use.go | base |
| read_file | tool_use.go | base |
| write_file | tool_use.go | base |
| edit_file | tool_use.go | base |
| task_create | task.go | s07 |
| task_list | task.go | s07 |
| task_get | task.go | s07 |
| task_update | task.go | s07 |
| task_bind_worktree | task.go | s12 |
| load_skill | skill.go | s05 |
| background_run | background.go | s08 |
| check_background | background.go | s08 |
| spawn_teammate | team/manager.go | s09 |
| list_teammates | team/manager.go | s09 |
| send_message | bus/bus.go | s09 |
| read_inbox | bus/bus.go | s09 |
| broadcast | bus/bus.go | s09 |
| shutdown_request | team/manager.go | s10 |
| claim_task | task.go | s11 |
| worktree_create | worktree/manager.go | s12 |
| worktree_list | worktree/manager.go | s12 |
| worktree_status | worktree/manager.go | s12 |
| worktree_run | worktree/manager.go | s12 |
| worktree_keep | worktree/manager.go | s12 |
| worktree_remove | worktree/manager.go | s12 |
| worktree_events | worktree/events.go | s12 |

---

## 5. 目录结构

```
project/
├── .team/
│   ├── config.json
│   └── inbox/
│       ├── alice.jsonl
│       ├── bob.jsonl
│       ├── lead.jsonl
├── .tasks/
│   ├── task_1.json
│   ├── task_2.json
├── .worktrees/
│   ├── index.json
│   ├── events.jsonl
│   ├── auth-refactor/
│   ├── feature-xyz/
├── skills/
│   ├── rust/SKILL.md
│   └── ...
└── agent/
    ├── tool_use.go
    ├── skill.go
    ├── task.go
    ├── background.go
    ├── team/
    │   ├── bus.go
    │   ├── config.go
    │   ├── member.go
    │   └── manager.go
    ├── worktree/
    │   ├── manager.go
    │   └── events.go
    └── agent_loop.go
```

---

## 6. 创建顺序

1. **tool_use.go** - 底层依赖
2. **task.go** - 任务管理核心
3. **skill.go** - 技能加载
4. **background.go** - 后台任务 (依赖 task.go)
5. **bus/bus.go** - Bus 接口定义
6. **bus/jsonl_bus.go** - JSONL 实现
7. **team/config.go** - 团队配置
8. **team/member.go** - Teammate 循环 (依赖 tool_use.go, bus/bus.go)
9. **team/manager.go** - 团队管理 (依赖 team/*, task.go)
10. **worktree/events.go** - 事件总线
11. **worktree/manager.go** - Worktree 管理 (依赖 task.go, events.go)
12. **agent_loop.go** - 主循环 (组合所有模块)

---

## 7. 未来扩展：Redis 实现

```go
// bus/redis_bus.go - 未来实现
type RedisBus struct {
    client *redis.Client
    prefix string
}

func NewRedisBus(addr, password string, db int) *RedisBus

func (b *RedisBus) Send(from, to, content, msgType string, extra map[string]any) string {
    // LPUSH to inbox list
    key := fmt.Sprintf("inbox:%s", to)
    msg := Message{Type: msgType, From: from, To: to, Content: content, Timestamp: time.Now().Unix(), Extra: extra}
    data, _ := json.Marshal(msg)
    b.client.LPush(context.Background(), key, data)
    return fmt.Sprintf("Sent %s to %s", msgType, to)
}

func (b *RedisBus) ReadInbox(name string) []Message {
    key := fmt.Sprintf("inbox:%s", name)
    // LRANGE + LTRIM to atomic read and clear
    vals, _ := b.client.LRange(context.Background(), key, 0, -1).Result()
    b.client.Del(context.Background(), key)
    
    var msgs []Message
    for _, v := range vals {
        var msg Message
        json.Unmarshal([]byte(v), &msg)
        msgs = append(msgs, msg)
    }
    return msgs
}
```

切换实现只需：
```go
var globalBus bus.Bus

func init() {
    impl := os.Getenv("BUS_IMPL") // "jsonl" or "redis"
    if impl == "redis" {
        globalBus = bus.NewRedisBus("localhost:6379", "", 0)
    } else {
        globalBus = bus.NewJSONLBus(WORKDIR)
    }
}
```