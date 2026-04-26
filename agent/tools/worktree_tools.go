package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	REPO_ROOT string
	WORKTREE *WorktreeManager
)

func init_worktree(workdir string) {
	var err error
	REPO_ROOT, err = detect_repo_root(workdir)
	//(TODO) err may be caused by git not being installed or the directory not being a git repo
	if err != nil {
		REPO_ROOT = workdir
	}
}

type Worktree struct {
    Name   string
    Path  string
    Branch string
    Active bool
}

// WorktreeManager 管理所有 worktree
type WorktreeManager struct {
    rootdir   string
    worktrees map[string]*Worktree
    current   string
    mu        sync.RWMutex
    bus       *EventBus// 内嵌 eventbus，不独立
}

type EventBus struct {
	
}



func detect_repo_root(workdir string) (string, error) {
	timeout := 10
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = workdir
	
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("Error: Timeout (%ds)", timeout)
	}
	if err != nil {
		return "", err
	}

	repo_root := strings.TrimSpace(string(output))
	
	return repo_root, nil
}

func NewWorktreeManager(workdir string) *WorktreeManager {
	
}
