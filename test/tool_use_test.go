package tester

import (
	"testing"
)

func TestToolBash(t *testing.T) {
	result := GetTestTools().Registry.Dispatch("bash", map[string]any{"command": "pwd"})
	if result != "" {
		t.Logf("success, got %s", result)
	}
}

func TestToolWriteAndRead(t *testing.T) {
	testFile := "test_test.txt"
	writeContent := "hello world\nthis is test content"

	writeResult := GetTestTools().Registry.Dispatch("write_file", map[string]any{
		"path":    testFile,
		"content": writeContent,
	})
	t.Logf("📝 写文件结果: %s", writeResult)

	readResult := GetTestTools().Registry.Dispatch("read_file", map[string]any{
		"path": testFile,
	})
	t.Logf("📖 读文件结果:\n%s", readResult)

	if readResult != writeContent {
		t.Fatalf("❌ 内容不匹配\n写入: %q\n读取: %q", writeContent, readResult)
	}

	t.Log("✅ 写 + 读 文件测试全部通过！")
}
