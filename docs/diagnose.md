## Debug Go SDK 的方法

### 1. 用 `go doc` 查看 SDK 文档

```bash
go doc github.com/anthropics/anthropic-sdk-go        # 包概览
go doc github.com/anthropics/anthropic-sdk-go MessageNewParams  # 特定类型
go doc github.com/anthropics/anthropic-sdk-go MessageNewParams.System  # 特定字段
```

### 2. 用 `go build` 看具体报错

```bash
go build ./agent/agent_loop.go
```

这会显示所有编译错误，比猜更高效。

### 3. 搜索文档的技巧

#### 用 codesearch 找示例代码
```bash
codesearch search --query "anthropic-sdk-go Messages.NewParams tool use example"
```

#### 直接看 GitHub 源码
```bash
webfetch --url "https://github.com/anthropics/anthropic-sdk-go/blob/main/tools.md"
```

#### 用 grep 快速定位
```bash
go doc github.com/anthropics/anthropic-sdk-go | grep -A 10 "StopReason"
```

---

## 调试思路（步骤）

### 步骤 1: 看报错 + go.mod 确认版本

```
anthropic-sdk-go v1.29.0
```

### 步骤 2: 看返回类型

```bash
go doc github.com/anthropics/anthropic-sdk-go NewClient  # 返回 Client 还是 *Client
```

### 步骤 3: 找对的构造方法

```bash
# 用 | grep 找 New 开头的构造方法
go doc github.com/anthropics/anthropic-sdk-go | grep "New"
# 结果: NewAssistantMessage, NewUserMessage, NewTextBlock, NewToolResultBlock...
```

### 步骤 4: 看字段类型

```bash
go doc github.com/anthropics/anthropic-sdk-go MessageNewParams.Tools
# Tools []ToolUnionParam  ← 不是 []ToolParam
```

### 步骤 5: 修复后立即 `go build` 验证

---

## 常用命令汇总

| 目的 | 命令 |
|------|------|
| 包概览 | `go doc <package>` |
| 类型详情 | `go doc <package> <Type>` |
| 字段类型 | `go doc <package> <Type>.<Field>` |
| 找构造方法 | `go doc <package> \| grep "New"` |
| 找常量 | `go doc <package> \| grep <ConstantName>` |
| 构建验证 | `go build <file>` |
| 找代码示例 | `codesearch search --query "<SDK> <usage>"` |

---

## agent_loop.go 修复过程

### 问题：代码使用旧版 API，与 SDK v1.29.0 不兼容

### 修复过程：

#### 1. go build 发现错误

```bash
$ go build ./agent/agent_loop.go

# command-line-arguments
agent/agent_loop.go:65:27: undefined: anthropic.F
agent/agent_loop.go:66:27: undefined: anthropic.F
agent/agent_loop.go:79:25: undefined: anthropic.Must
agent/agent_loop.go:80:25: undefined: anthropic.F
agent/agent_loop.go:82:15: cannot use tools (variable of type []anthropic.ToolParam) as []anthropic.ToolUnionParam value in struct literal
agent/agent_loop.go:83:25: undefined: anthropic.F
agent/agent_loop.go:91:40: undefined: anthropic.ContentBlockUnionParam
...
```

主要问题：
- `anthropic.F` 未定义（旧版 API）
- `anthropic.Must` 未定义
- `ToolParam` 类型错误，应为 `ToolUnionParam`
- `ContentBlockUnionParam` 未定义

#### 2. 确认 SDK 版本

```bash
$ cat go.mod | grep anthropic

github.com/anthropics/anthropic-sdk-go v1.29.0
```

#### 3. 查看 Messages.NewParams 的正确用法

```bash
go doc github.com/anthropics/anthropic-sdk-go MessageNewParams
```

输出显示 `System` 字段是 `[]TextBlockParam` 类型，不是旧版的单个对象。

#### 4. 查看 Tools 字段的正确类型

```bash
go doc github.com/anthropics/anthropic-sdk-go MessageNewParams.Tools
```

发现应使用 `[]ToolUnionParam`，不是 `[]ToolParam`。

#### 5. 搜索 ToolUnionParam 的构造方法

```bash
go doc github.com/anthropics/anthropic-sdk-go ToolUnionParam | grep "func"
```

发现 `ToolUnionParamOfTool(inputSchema ToolInputSchemaParam, name string)` 是正确的构造方法。

#### 6. 查找 New 开头的构造方法

```bash
go doc github.com/anthropics/anthropic-sdk-go | grep "New"
```

发现：
- `NewAssistantMessage(blocks ...ContentBlockParamUnion) MessageParam`
- `NewUserMessage(blocks ...ContentBlockParamUnion) MessageParam`
- `NewTextBlock(text string) ContentBlockParamUnion`
- `NewToolResultBlock(toolUseID string, content string, isError bool) ContentBlockParamUnion`
- `NewToolUseBlock(id string, input any, name string) ContentBlockParamUnion`

#### 7. 确认 StopReason 常量

```bash
go doc github.com/anthropics/anthropic-sdk-go StopReasonEndTurn
```

输出：
```go
const (
    StopReasonEndTurn      StopReason = "end_turn"
    StopReasonMaxTokens    StopReason = "max_tokens"
    StopReasonStopSequence StopReason = "stop_sequence"
    StopReasonToolUse      StopReason = "tool_use"
    StopReasonPauseTurn    StopReason = "pause_turn"
    StopReasonRefusal      StopReason = "refusal"
)
```

#### 8. 确认 NewClient 返回值类型

```bash
go doc github.com/anthropics/anthropic-sdk-go NewClient
```

返回 `Client`，不是 `*Client`。

#### 9. 验证 GetText 方法签名

```bash
go doc github.com/anthropics/anthropic-sdk-go ContentBlockParamUnion.GetText
```

返回 `*string`，不是 `(string, bool)` 元组。

---

## 修复后的代码对比

### System 字段

```go
# 旧版（错误）
System: anthropic.F([]anthropic.TextBlockParam{{Type: "text", Text: anthropic.F(system)}}),

# 新版（正确）
System: []anthropic.TextBlockParam{
    {Text: system},
},
```

### Tools 字段

```go
# 旧版（错误）
tools := []anthropic.ToolParam{...}
Tools: tools,

# 新版（正确）
Tools: []anthropic.ToolUnionParam{
    anthropic.ToolUnionParamOfTool(
        anthropic.ToolInputSchemaParam{...},
        "bash",
    ),
},
```

### 消息创建

```go
# 旧版（错误）
*messages = append(*messages, anthropic.MessageParam{
    Role:    anthropic.F("assistant"),
    Content: assistantContent,
})

# 新版（正确）
*messages = append(*messages, anthropic.NewAssistantMessage(assistantContent...))
```

### StopReason

```go
# 旧版（错误）
if resp.StopReason != anthropic.MessageStopReasonToolUse {

# 新版（正确）
if resp.StopReason != anthropic.StopReasonToolUse {
```

