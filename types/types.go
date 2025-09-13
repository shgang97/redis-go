package types

// 命令类型
const (
	CmdUnknown = iota
	CmdPing
	CmdSet
	CmdGet
	CmdDel
	CmdQuit
)

// Command 解析后的命令结构
type Command struct {
	Cmd  int
	Args []string
}
