package tools

var WORKDIR string

func Init_tools(workdir string) {
	WORKDIR = workdir
	init_todo_manager()
	init_task_manager(workdir)
	init_skills(workdir)
	init_worktree(workdir)
}
