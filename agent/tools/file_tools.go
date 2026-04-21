package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RunRead(path string, workdir string, limit int) string {
	safePath, err := SafePath(path, workdir)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	text := string(content)
	if len(text) > 50000 {
		text = text[:50000]
	}

	lines := strings.Split(text, "\n")
	if limit > 0 && limit < len(lines) {
		lines = append(lines[:limit], fmt.Sprintf("... (%d more lines)", len(lines)-limit))
	}

	return strings.Join(lines, "\n")
}

func RunWrite(path string, content string, workdir string) string {
	safePath, err := SafePath(path, workdir)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if err := os.WriteFile(safePath, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return fmt.Sprintf("Wrote %d bytes to %s", len(content), path)
}

func RunEdit(path string, oldText string, newText string, workdir string) string {
	safePath, err := SafePath(path, workdir)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, oldText) {
		return fmt.Sprintf("Error: Text not found in %s", path)
	}

	newContent := strings.Replace(text, oldText, newText, 1)
	if err := os.WriteFile(safePath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return fmt.Sprintf("Edited %s", path)
}
