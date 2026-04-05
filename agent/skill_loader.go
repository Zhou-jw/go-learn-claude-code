/*
 * 1. What skills are available?
 * 2. Load the agent-builder skill and follow its instructions
 * 3. I need to do a code review -- load the relevant skill first
 * 4. Build an MCP server using the mcp-builder skill
 */
 
package agent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// 全局单例（对应 Python 的 SKILL_LOADER = SkillLoader(...)）
var SKILL_LOADER *SkillLoader

// 初始化全局 loader
func init() {
	skillsDir, _ := os.Getwd()
	skillsDir = filepath.Join(skillsDir, "skills")
	if skillsDir == "" {
		skillsDir = "skills"
	}
	fmt.Printf("skillsDir is %s\n", skillsDir)
	SKILL_LOADER = NewSkillLoader(skillsDir)
}

// Skill 对应 Python 里的 skill 结构
type Skill struct {
	Meta map[string]any `yaml:"-"`
	Body string         `yaml:"-"`
	Path string         `yaml:"-"`
}

// SkillLoader 对应 Python 的 SkillLoader
type SkillLoader struct {
	skillsDir string
	Skills    map[string]*Skill
}

// NewSkillLoader 构造函数
func NewSkillLoader(skillsDir string) *SkillLoader {
	sl := &SkillLoader{
		skillsDir: skillsDir,
		Skills:    make(map[string]*Skill),
	}
	sl.loadAll()
	return sl
}

// loadAll 加载所有 skills/*/SKILL.md
func (sl *SkillLoader) loadAll() {
	// 目录不存在直接返回
	if _, err := os.Stat(sl.skillsDir); os.IsNotExist(err) {
		return
	}

	var files []string

	// 递归查找 **/SKILL.md
	err := filepath.Walk(sl.skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "SKILL.md" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return
	}

	// 排序（和 Python sorted 一致）
	sort.Strings(files)

	// 逐个解析
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		meta, body := sl.ParseFrontmatter(string(content))
		parentDir := filepath.Base(filepath.Dir(path))
		name, ok := meta["name"].(string)
		if !ok || name == "" {
			name = parentDir
		}

		sl.Skills[name] = &Skill{
			Meta: meta,
			Body: body,
			Path: path,
		}
	}
}

// ParseFrontmatter 解析 --- yaml ---\n内容
func (sl *SkillLoader) ParseFrontmatter(text string) (map[string]any, string) {
	// 正则匹配：^---\n(.*?)\n---\n(.*)
	re := regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)`)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return map[string]any{}, strings.TrimSpace(text)
	}

	metaYaml := match[1]
	body := strings.TrimSpace(match[2])

	var meta map[string]any
	if err := yaml.Unmarshal([]byte(metaYaml), &meta); err != nil {
		return map[string]any{}, body
	}

	if meta == nil {
		meta = map[string]any{}
	}
	return meta, body
}

// GetDescriptions 生成技能摘要（给 system prompt）
func (sl *SkillLoader) GetDescriptions() string {
	if len(sl.Skills) == 0 {
		return "(no skills available)"
	}

	// 按名称排序
	names := make([]string, 0, len(sl.Skills))
	for name := range sl.Skills {
		names = append(names, name)
	}
	sort.Strings(names)

	var lines []string
	for _, name := range names {
		skill := sl.Skills[name]
		desc, _ := skill.Meta["description"].(string)
		if desc == "" {
			desc = "No description"
		}
		tags, _ := skill.Meta["tags"].(string)

		line := fmt.Sprintf("  - %s: %s", name, desc)
		if tags != "" {
			line += fmt.Sprintf(" [%s]", tags)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// GetContent 获取完整技能内容
func (sl *SkillLoader) GetContent(input map[string]any) string {
	name, ok := input["name"].(string)
	if !ok {
		return "Error: name is required"
	}
	
	skill, ok := sl.Skills[name]
	if !ok {
		available := make([]string, 0, len(sl.Skills))
		for k := range sl.Skills {
			available = append(available, k)
		}
		return fmt.Sprintf("Error: Unknown skill '%s'. Available: %s", name, strings.Join(available, ", "))
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("<skill name=\"%s\">\n", name))
	buf.WriteString(skill.Body)
	buf.WriteString("\n</skill>")
	return buf.String()
}
