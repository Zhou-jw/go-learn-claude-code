# Worktree 设计文档

## 1. 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                     Agent Loop                              │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │ TaskManager │    │WorktreeManager│    │  EventBus   │  │
│  │  (.task/)   │    │ (.worktrees/) │    │ events.jsonl│  │
│  └─────────────┘    └──────────────┘    └──────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
                 ┌─────────────────┐
                 │  Git Worktrees  │
                 │  - auth-refactor│
                 │  - ui-cleanup   │
                 └─────────────────┘
```

## 2. 数据目录结构

```
.worktrees/                  # worktree持久化目录
├── index.json              # worktree索引（崩溃恢复用）
├── events.jsonl            # 事件日志（追加写入）
└── worktrees/              # 各worktree目录
    ├── auth-refactor/
    │   └── .git            # worktree引用
    └── ui-cleanup/
        └── .git

.tasks/                      # 任务持久化目录（已有）
├── task_1.json
├── task_2.json
└── ...
```

## 3. 事件流设计

### 3.1 事件类型

| 事件 | 说明 | 触发时机 |
|------|------|----------|
| `worktree.create.before` | 创建前 | 执行`git worktree add`前 |
| `worktree.create.after` | 创建后 | worktree创建成功 |
| `worktree.create.failed` | 创建失败 | 创建过程中出错 |
| `worktree.remove.before` | 删除前 | 执行`git worktree remove`前 |
| `worktree.remove.after` | 删除后 | worktree删除成功 |
| `worktree.remove.failed` | 删除失败 | 删除过程中出错 |
| `worktree.keep` | 保留 | 任务完成但保留worktree |
| `task.completed` | 任务完成 | 关联任务状态变为done |

### 3.2 事件格式 (events.jsonl)

```json
{
  "event": "worktree.create.after",
  "task": {"id": 1, "status": "running"},
  "worktree": {"name": "auth-refactor", "path": "/path/to/worktrees/auth-refactor", "branch": "auth-refactor"},
  "ts": 1730000000
}
```

### 3.3 事件流示例

```
用户创建任务 #1: "重构auth模块"
    │
    ▼
worktree.create.before  ──写入事件──▶ events.jsonl
    │
    ▼
git worktree add ../worktrees/auth-refactor -b auth-refactor
    │
    ├─ 成功 ──▶ worktree.create.after ──▶ events.jsonl
    │                    │
    │                    ▼
    │              更新 index.json
    │
    └─ 失败 ──▶ worktree.create.failed ──▶ events.jsonl
                            │
                            ▼
                      写入错误信息
```

## 4. 核心模块设计

### 4.1 EventBus (扩展现有 bus/)

```go
// agent/team/bus/event_bus.go

type WorktreeEvent struct {
    Event    string          `json:"event"`
    Task     *Task           `json:"task,omitempty"`
    Worktree *WorktreeState  `json:"worktree,omitempty"`
    TS       int64           `json:"ts"`
}

type EventBus interface {
    // 记录事件到 events.jsonl
    Emit(event WorktreeEvent) error

    // 读取最近的事件（用于恢复）
    RecentEvents(limit int) []WorktreeEvent
}
```

### 4.2 WorktreeManager

```go
// agent/tools/worktree/manager.go

type WorktreeManager struct {
    rootDir string           // 仓库根目录
    wtDir  string           // worktrees目录
    index  *WorktreeIndex   // 内存中的索引
    bus    EventBus         // 事件总线
    mu     sync.RWMutex
}

type WorktreeIndex struct {
    Worktrees map[string]*WorktreeState `json:"worktrees"`
    UpdatedAt int64                     `json:"updated_at"`
}

type WorktreeState struct {
    Name      string    `json:"name"`
    Path      string    `json:"path"`
    Branch    string    `json:"branch"`
    TaskID    int       `json:"task_id"`
    Status    string    `json:"status"`  // creating/active/removing/removed
    CreatedAt int64     `json:"created_at"`
}
```

### 4.3 关键方法

```go
func (wm *WorktreeManager) Create(name, branch string, taskID int) error
func (wm *WorktreeManager) Remove(name string) error
func (wm *WorktreeManager) Keep(name string) error
func (wm *WorktreeManager) List() []*WorktreeState
func (wm *WorktreeManager) Get(name string) *WorktreeState
```

## 5. 崩溃恢复流程

### 5.1 恢复触发条件

- Agent启动时检测到`.worktrees/index.json`存在
- 或`.tasks/`目录有未完成的任务

### 5.2 恢复步骤

```
1. 加载 .worktrees/index.json
       │
       ▼
2. 遍历所有 active 状态的 worktree
       │
       ▼
3. 验证目录是否存在
       ├─ 存在 ──▶ 恢复内存状态
       │
       └─ 不存在 ──▶ 标记为 lost，写入恢复日志
       │
       ▼
4. 检查 .tasks/ 中未完成的任务
       │
       ▼
5. 重新关联任务与 worktree
```

### 5.3 恢复代码示意

```go
func (wm *WorktreeManager) Recover() {
    // 1. 加载索引
    index := wm.loadIndex()

    // 2. 验证并恢复
    for name, wt := range index.Worktrees {
        if wt.Status == "active" {
            if _, err := os.Stat(wt.Path); err == nil {
                wm.index.Worktrees[name] = wt
            } else {
                // 目录丢失，标记状态
                wt.Status = "lost"
                wm.Emit(WorktreeEvent{
                    Event: "worktree.recover.failed",
                    Worktree: wt,
                    TS: time.Now().Unix(),
                })
            }
        }
    }
}
```

## 6. 事件与任务关联

### 6.1 设计原则

- **Task是主体**: 任务描述工作内容
- **Worktree是容器**: worktree为任务提供隔离环境
- **生命周期绑定**: 任务完成时决定worktree是删除还是保留

### 6.2 状态流转

```
Task: pending → running → done
         │         │
         ▼         ▼
Worktree: 创建 → active → (remove/keep)
```

### 6.3 Task结构扩展

```go
type Task struct {
    ID          int       `json:"id"`
    Subject     string    `json:"subject"`
    State       TaskState `json:"state"`
    Worktree    string    `json:"worktree,omitempty"`  // 关联的worktree名称
    // ... 其他字段
}
```

## 7. 工具函数 (Tool Schema)

### 7.1 新增工具

```json
{
  "name": "worktree_create",
  "description": "为任务创建独立的git worktree",
  "input_schema": {
    "type": "object",
    "properties": {
      "name": {"type": "string", "description": "worktree名称"},
      "branch": {"type": "string", "description": "分支名（默认与name相同）"},
      "task_id": {"type": "integer", "description": "关联的任务ID"}
    },
    "required": ["name", "task_id"]
  }
}

{
  "name": "worktree_remove",
  "description": "删除worktree",
  "input_schema": {
    "type": "object",
    "properties": {
      "name": {"type": "string", "description": "worktree名称"},
      "force": {"type": "boolean", "description": "强制删除"}
    },
    "required": ["name"]
  }
}

{
  "name": "worktree_list",
  "description": "列出所有worktree"
}

{
  "name": "worktree_keep",
  "description": "保留worktree不删除",
  "input_schema": {
    "type": "object",
    "properties": {
      "name": {"type": "string", "description": "worktree名称"}
    },
    "required": ["name"]
  }
}
```

## 8. 实现步骤

### Phase 1: 基础架构
1. 扩展 `team/bus/` 添加 WorktreeEvent 类型
2. 实现 `EventBus` 写入 `.worktrees/events.jsonl`
3. 完成 `WorktreeManager` 核心方法

### Phase 2: 生命周期
1. 实现 `Create`/`Remove`/`Keep` 方法
2. 集成事件触发（before/after/failed）
3. 更新 `index.json` 持久化

### Phase 3: 崩溃恢复
1. 实现 `Recover()` 方法
2. 启动时自动检测并恢复
3. 写恢复日志到events

### Phase 4: 工具集成
1. 在 `tool_handler.go` 注册处理函数
2. 在 `tool_schema.go` 添加工具定义
3. 与 TaskManager 关联

## 9. 代码组织

```
agent/
├── team/
│   └── bus/
│       ├── bus.go          # 现有 Bus 接口
│       ├── jsonl_bus.go     # 现有实现
│       ├── event_bus.go     # 新增: WorktreeEventBus
│       └── events.go        # 新增: 事件类型定义
├── tools/
│   ├── worktree_tools.go   # 现有框架
│   └── worktree/           # 新目录
│       ├── manager.go      # WorktreeManager
│       ├── git.go          # Git操作封装
│       ├── recover.go     # 恢复逻辑
│       └── handler.go     # 工具处理函数
```

## 10. 注意事项

1. **事务性**: 事件写入应在操作前后都触发，确保完整追踪
2. **幂等性**: 恢复时去重，避免重复处理
3. **清理**: 定期清理过期的events.jsonl（可选）
4. **错误处理**: git命令超时要有明确日志