package tools

import "sync"

type Worktree struct {
    Name   string
    Path  string
    Branch string
    Active bool
}

// WorktreeManager 管理所有 worktree
type WorktreeManager struct {
    rootDir   string
    worktrees map[string]*Worktree
    current   string
    mu        sync.RWMutex
    bus       *EventBus// 内嵌 eventbus，不独立
}

type EventBus struct {
	
}

func init_worktree(workdir string) {
	
}