# Worktree 与 EventBus 组件设计指南（修正版）

> 基于 `/home/code/go-learn-claude-code/agent` 代码分析，参考 `/home/code/opencode-dev` 架构设计

## 1. opencode/claude-code 架构分析

### 1.1 worktree 实现（TypeScript/Effect）

```typescript
// packages/opencode/src/worktree/index.ts
// 核心：Effect Service 模式

export interface Interface {
  readonly makeWorktreeInfo: (name?: string) => Effect.Effect<Info>
  readonly createFromInfo: (info: Info, startCommand?: string) => Effect.Effect<void>
  readonly create: (input?: CreateInput) => Effect.Effect<Info>
  readonly remove: (input: RemoveInput) => Effect.Effect<boolean>
  readonly reset: (input: ResetInput) => Effect.Effect<boolean>
}

export class Service extends Context.Service<Service, Interface>()("@opencode/Worktree") {}
```

特点：**内嵌事件发布**，不依赖独立 eventbus 层

```typescript
// 内部直接发布事件
GlobalBus.emit("event", {
  directory: info.directory,
  project: ctx.project.id,
  workspace: workspaceID,
  payload: { type: Event.Ready.type, properties: { name: info.name, branch: info.branch } },
})
```

### 1.2 bus 实现（两种）

| 组件 | 用途 | 位置 |
|------|------|------|
| GlobalBus | 跨进程/全局通信 | `bus/global.ts` (EventEmitter) |
| Bus.Service | 进程内通信 | `bus/index.ts` (Effect PubSub) |

---

## 2. 你的代码架构分析

### 2.1 现有结构

```
agent/
├── tools/              # 工具层：bash/file/todo/task/skill
├── team/               # team 层
│   └── bus/
│       ├── bus.go      # Bus 接口
│       └── jsonl_bus.go # team 消息队列
```

### 2.2 关键发现

**jsonl_bus 只给 team 用，eventbus 是给 worktree 用**——不应该抽象出独立 bus 层！

- jsonl_bus: 持久化消息队列（team 成员间通信）
- eventbus: 内存事件总线（worktree 内部事件通知）

---

## 3. Worktree 组件设计（Go 版本）

### 3.1 成员变量设计

```go
// agent/tools/worktree.go

// Worktree 表示一个工作树
type Worktree struct {
    Name   string
    Path  string
    Branch string
    Active bool
}

// WorktreeManager 管理所有 worktree
type WorktreeManager struct {
    rootDir   string
    worktrees map[string]*Worktree
    current   string
    mu        sync.RWMutex
    bus       *WorktreeEventBus // 内嵌 eventbus，不独立
}

// WorktreeEventBus 内嵌在 worktree 模块内
type WorktreeEventBus struct {
    subscribers map[string][]EventHandler
    mu          sync.RWMutex
    queue       chan Event
}
```

### 3.2 方法设计

```go
// 核心 CRUD
func (m *WorktreeManager) Create(name, branch string) (*Worktree, error)
func (m *WorktreeManager) Delete(name string) error
func (m *WorktreeManager) List() []*Worktree
func (m *WorktreeManager) Switch(name string) error
func (m *WorktreeManager) Run(name, cmd string, timeout int64) (string, error)

// eventbus 内嵌方法
func (eb *WorktreeEventBus) Subscribe(eventType string, handler EventHandler)
func (eb *WorktreeEventBus) Publish(event Event)
```

### 3.3 事件类型定义

```go
// agent/tools/worktree_event.go (与 worktree.go 同包)

const (
    EventWorktreeReady  = "worktree.ready"
    EventWorktreeError = "worktree.error"
)

type WorktreeEvent struct {
    Type      string
    Payload  interface{}
    Source   string
}
```

---

## 4. 文件组织（修正）

### 4.1 不提取独立 bus 层

错误设计（之前）：
```
agent/
├── bus/          # 独立总线层（错误）
│   ├── event_bus.go
│   └── event_types.go
```

正确设计：
```
agent/
├── tools/               # 工具层
│   ├── worktree.go      # worktree 管理器（含 eventbus）
│   ├── worktree_event.go # 事件类型
│   ├── worktree_tools.go
│   └── worktree_handler.go
├── team/                # team 层
│   ├── bus/
│   │   ├── bus.go
│   │   └── jsonl_bus.go
```

### 4.2 文件清单

| 文件 | 功能 |
|------|------|
| `tools/worktree.go` | WorktreeManager + WorktreeEventBus |
| `tools/worktree_event.go` | 事件类型常量 + Event 结构体 |
| `tools/worktree_tools.go` | 工具 schema |
| `tools/worktree_handler.go` | handler 实现 |

---

## 5. 集成方式

### 5.1 在 init 中初始化

```go
// agent/tools/tool_init.go

func Init_tools(workdir string) {
    WORKDIR = workdir
    init_todo_manager()
    init_task_manager(workdir)
    init_skills(workdir)
    init_worktree_manager(workdir) // 新增
}
```

### 5.2 在 handler 中注册

```go
// agent/tools/tool_handler.go

var TOOL_HANDLERS = map[string]ToolHandler{
    // 现有...
    "worktree_create": handleWorktreeCreate,
    "worktree_list":   handleWorktreeList,
    "worktree_switch": handleWorktreeSwitch,
    "worktree_run":   handleWorktreeRun,
}
```

---

## 6. 设计原则总结

| 原则 | 说明 |
|------|------|
| **不提取独立 bus 层** | eventbus 内嵌到使用它的组件（如 worktree） |
| **按领域分组** | team 用 jsonl_bus，worktree 用内嵌 eventbus |
| **内嵌发布** | 组件内部直接发布事件，不依赖全局 bus |
| **复用现有模式** | Manager-Task 模式 + Handler 注册模式 |

---

> 文档修正时间：2024
> 参考：
> - `/home/code/go-learn-claude-code/agent/`
> - `/home/code/opencode-dev/packages/opencode/src/worktree/index.ts`