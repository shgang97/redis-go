package protocol

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/shgang97/redis-go/types"
)

// ParseCommand 解析RESP命令
func ParseCommand(raw []byte) *types.Command {
	parts := bytes.Split(bytes.TrimSpace(raw), []byte(" "))
	if len(parts) == 0 {
		return &types.Command{
			Cmd: types.CmdUnknown,
		}
	}

	cmd := strings.ToUpper(string(parts[0]))
	args := make([]string, 0)
	for i := 1; i < len(parts); i++ {
		args = append(args, string(parts[i]))
	}
	switch cmd {
	case "PING":
		return &types.Command{types.CmdPing, args}
	case "SET":
		return &types.Command{types.CmdSet, args}
	case "GET":
		return &types.Command{types.CmdGet, args}
	case "DEL":
		return &types.Command{types.CmdDel, args}
	case "QUIT":
		return &types.Command{types.CmdQuit, args}
	default:
		return &types.Command{types.CmdUnknown, args}
	}
}

// FormatReply 格式化回复
func FormatReply(value string) []byte {
	if value == "" {
		return []byte("$-1\r\n")
	}
	return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(value), value))
}

// FormatError 格式化错误
func FormatError(msg string) []byte {
	return []byte(fmt.Sprintf("-%s\r\n", msg))
}

// formatSimpleString
func FormatSimpleString(msg string) []byte {
	return []byte(fmt.Sprintf("+%s\r\n", msg))
}
