package tools

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type BashConfig struct {
	DangerousCommands []string
	Workdir           string
}

func NewBashConfig(workdir string) *BashConfig {
	return &BashConfig{
		Workdir: workdir,
		DangerousCommands: []string{
			"rm -rf /",
			"sudo",
			"shutdown",
			"reboot",
			"> /dev/",
		},
	}
}

type ToolError struct {
	msg string
}

func (e *ToolError) Error() string {
	return e.msg
}

func NewToolError(format string, args ...any) *ToolError {
	return &ToolError{msg: fmt.Sprintf(format, args...)}
}

func SafePath(p string, workdir string) (string, error) {
	absPath := filepath.Join(workdir, p)
	absPath = filepath.Clean(absPath)
	workDirAbs, _ := filepath.Abs(workdir)
	if !strings.HasPrefix(absPath, workDirAbs) {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}
	return absPath, nil
}

func RunBashWithConfig(cfg *BashConfig, command string, timeout int64) (string, error) {
	for _, d := range cfg.DangerousCommands {
		if strings.Contains(command, d) {
			return "", fmt.Errorf("Error: Dangerous command blocked")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = cfg.Workdir

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("Error: Timeout (%ds)", timeout)
	}
	if err != nil {
		return "", err
	}

	out := strings.TrimSpace(string(output))
	if out == "" {
		out = "(no output)"
	}
	if len(out) > 50000 {
		out = out[:50000] + "...\n"
	}
	return out, nil
}

func RunBash(command string, timeout int64, workdir string) (string, error) {
	cfg := NewBashConfig(workdir)
	return RunBashWithConfig(cfg, command, timeout)
}
