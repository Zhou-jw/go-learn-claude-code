package tester

import (
	// "fmt"
	"glcc/agent/tools"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 测试前初始化临时技能目录
func setupTestDir() string {
	root := "../skills"
	return root
}

// 测试：加载所有技能
func TestSkillLoader_LoadAll(t *testing.T) {
	dir := setupTestDir()
	loader := tools.NewSkillLoader(dir)

	// 应该加载 4 个技能
	assert.Len(t, loader.Skills, 4)

	// 检查存在性
	assert.NotNil(t, loader.Skills["pdf"])
	assert.NotNil(t, loader.Skills["mcp-builder"])
	assert.NotNil(t, loader.Skills["code-review"])
	assert.NotNil(t, loader.Skills["agent-builder"])
}

// 测试：解析 yaml frontmatter
func TestSkillLoader_ParseFrontmatter(t *testing.T) {
	loader := &tools.SkillLoader{}
	text := `---
			name: test
			desc: hello
			---
			body content`

	meta, body := loader.ParseFrontmatter(text)
	assert.Equal(t, "test", meta["name"])
	assert.Equal(t, "hello", meta["desc"])
	assert.Equal(t, "body content", body)

	// 无 frontmatter
	meta2, body2 := loader.ParseFrontmatter("only body")
	assert.Empty(t, meta2)
	assert.Equal(t, "only body", body2)
}

// 测试：获取技能描述列表
func TestSkillLoader_GetDescriptions(t *testing.T) {
	dir := setupTestDir()
	loader := tools.NewSkillLoader(dir)
	desc := loader.GetDescriptions()

	// fmt.Println(desc)
	// 包含所有技能
	assert.Contains(t, desc, "agent-builder: Design and build AI agents for any domain.")
	assert.Contains(t, desc, "code-review: Perform thorough code reviews with security, performance, and maintainability analysis.")
	assert.Contains(t, desc, "mcp-builder: Build MCP (Model Context Protocol) servers that give Claude new capabilities.")
	assert.Contains(t, desc, "pdf: Process PDF files")

}

// 测试：获取单个技能内容
func TestSkillLoader_GetContent(t *testing.T) {
	dir := setupTestDir()
	loader := tools.NewSkillLoader(dir)

	// 正常获取
	content := loader.GetContent("pdf")
	assert.Contains(t, content, "<skill name=\"pdf\">")
	// fmt.Println(content)
	// first line should be "# PDF Processing Skill"
	assert.Contains(t, content, "# PDF Processing Skill")
	// last line
	assert.Contains(t, content, "4. **OCR for scanned PDFs**: Use `pytesseract` if text extraction returns empty")

	// 不存在的技能
	errContent := loader.GetContent("not_exist")
	assert.Contains(t, errContent, "Error: Unknown skill 'not_exist'")
	// assert.Contains(t, errContent, "agent-builder, code-review, mcp-builder, pdf")

	// 无 name 参数
	errNoName := loader.GetContent("")
	assert.Equal(t, "Error: name is required", errNoName)
}

// 测试：目录不存在时返回空
func TestSkillLoader_DirNotExist(t *testing.T) {
	loader := tools.NewSkillLoader("not_exist_dir")
	assert.Empty(t, loader.Skills)
	assert.Equal(t, "(no skills available)", loader.GetDescriptions())
}
