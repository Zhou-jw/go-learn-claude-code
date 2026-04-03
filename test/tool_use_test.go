package tester

import (
	"glcc/agent"
	"testing"
)

func TestToolBash(t *testing.T) {
	result := agent.DispatchTool("bash", map[string]any{"command":"pwd"})
	if result != "" {
		t.Logf("success, got %s", result)
	}
}

func TestToolWriteAndRead(t *testing.T) {
	// 1. 定义测试用的文件名和内容
	testFile := "test_test.txt"
	writeContent := "hello world\nthis is test content"

	// 2. 调用 write_file 工具（写文件）
	writeResult := agent.DispatchTool("write_file", map[string]any{
		"path":    testFile,
		"content": writeContent,
	})
	t.Logf("📝 写文件结果: %s", writeResult)


	// 3. 调用 read_file 工具（读文件）
	readResult := agent.DispatchTool("read_file", map[string]any{
		"path": testFile,
	})
	t.Logf("📖 读文件结果:\n%s", readResult)


	// 4. 验证读到的内容和写入的一致
	if readResult != writeContent {
		t.Fatalf("❌ 内容不匹配\n写入: %q\n读取: %q", writeContent, readResult)
	}

	// 全部成功
	t.Log("✅ 写 + 读 文件测试全部通过！")
}