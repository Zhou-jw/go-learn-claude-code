# go-learn-claude-code

使用 Go 调用 Claude API 的学习项目

---

## 快速开始

### 1. 复制配置文件

```bash
cp config/example.yaml config/config.yaml
```

### 2. 编辑 config.yaml，填入你的 API Key

```yaml
anthropic:
  api_key: "sk-ant-xxxxx"  # 填入你的 API Key
```

### 3. 运行

```bash
go build && ./glcc
```

---

## 配置说明

| 文件 | 用途 |
|------|------|
| `example.yaml` | 配置模板，包含所有可用配置项 |
| `config.yaml` | 实际配置文件，包含你的 API Key（已加入 .gitignore）|

---
