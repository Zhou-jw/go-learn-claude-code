package tester

import (
	"os"
	"path/filepath"
	"testing"

	"glcc/agent/tools"

	"github.com/stretchr/testify/assert"
)

func setupTestToolsDir(t *testing.T) string {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)
	return tmpDir
}

func TestSafePath(t *testing.T) {
	workdir := setupTestToolsDir(t)

	// Valid path
	path, err := tools.SafePath("test.txt", workdir)
	assert.NoError(t, err)
	assert.Contains(t, path, "test.txt")

	// Path with subdirectory
	path, err = tools.SafePath("subdir/test.txt", workdir)
	assert.NoError(t, err)
	assert.Contains(t, path, "subdir")

	// Escape attempt
	_, err = tools.SafePath("../test.txt", workdir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path escapes workspace")
}

func TestRunBash(t *testing.T) {
	workdir := setupTestToolsDir(t)

	// Basic command
	result, err := tools.RunBash("echo hello", 10, workdir)
	assert.NoError(t, err)
	assert.Contains(t, result, "hello")

	// Dangerous command blocked
	_, err = tools.RunBash("rm -rf /", 10, workdir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Dangerous command blocked")
}

func TestRunRead(t *testing.T) {
	workdir := setupTestToolsDir(t)

	// Write a test file
	testFile := filepath.Join(workdir, "test.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3"), 0644)

	// Read full file
	result := tools.RunRead("test.txt", workdir, 0)
	assert.Contains(t, result, "line1")
	assert.Contains(t, result, "line3")

	// Read with limit
	result = tools.RunRead("test.txt", workdir, 2)
	assert.Contains(t, result, "line1")
	assert.Contains(t, result, "line2")
	assert.Contains(t, result, "more lines")
}

func TestRunWrite(t *testing.T) {
	workdir := setupTestToolsDir(t)

	// Write new file
	result := tools.RunWrite("newfile.txt", "hello world", workdir)
	assert.Contains(t, result, "Wrote")

	// Verify file content
	content, _ := os.ReadFile(filepath.Join(workdir, "newfile.txt"))
	assert.Equal(t, "hello world", string(content))

	// Write to subdirectory
	result = tools.RunWrite("subdir/nested.txt", "nested content", workdir)
	assert.Contains(t, result, "Wrote")

	content, _ = os.ReadFile(filepath.Join(workdir, "subdir", "nested.txt"))
	assert.Equal(t, "nested content", string(content))
}

func TestRunEdit(t *testing.T) {
	workdir := setupTestToolsDir(t)

	// Write initial file
	testFile := filepath.Join(workdir, "edit_test.txt")
	os.WriteFile(testFile, []byte("hello world"), 0644)

	// Edit file
	result := tools.RunEdit("edit_test.txt", "world", "go", workdir)
	assert.Contains(t, result, "Edited")

	// Verify change
	content, _ := os.ReadFile(testFile)
	assert.Equal(t, "hello go", string(content))

	// Edit non-existent text
	result = tools.RunEdit("edit_test.txt", "not exist", "fail", workdir)
	assert.Contains(t, result, "Error")
}

func TestRunEdit_PathEscape(t *testing.T) {
	workdir := setupTestToolsDir(t)

	result := tools.RunEdit("../escape.txt", "old", "new", workdir)
	assert.Contains(t, result, "Error")
}
