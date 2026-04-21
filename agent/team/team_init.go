package team

var TEAMMATE_MGR *TeammateManager

func Init_teammate_manager(workdir string) {
	TEAMMATE_MGR = NewTeammateManager(workdir)
}