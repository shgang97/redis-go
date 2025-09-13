package handler

import (
	"bytes"
	"fmt"

	callback "github.com/shgang97/redis-go/callbacks"
	client2 "github.com/shgang97/redis-go/client"
	"github.com/shgang97/redis-go/protocol"
	"github.com/shgang97/redis-go/types"

	"golang.org/x/sys/unix"
)

// RedisCallbackHandler 实现 EventCallbacks 接口
type RedisCallbackHandler struct {
}

func (h *RedisCallbackHandler) OnAccept(fd int, sa unix.Sockaddr, ctx callback.ServerContext) {
	// 设置非阻塞
	if err := unix.SetNonblock(fd, true); err != nil {
		fmt.Printf("SetNonblock err: %v\n", err)
		unix.Close(fd)
		return
	}

	// 注册事件
	ev := []unix.Kevent_t{
		{
			Ident:  uint64(fd),
			Filter: unix.EVFILT_READ,
			Flags:  unix.EV_ADD,
			Fflags: 0,
			Data:   0,
			Udata:  nil,
		},
	}
	kq := ctx.GetKqueue()
	if _, err := unix.Kevent(kq, ev, nil, nil); err != nil {
		fmt.Printf("Event err: %v\n", err)
		unix.Close(fd)
		return
	}

	// 创建客户端实例并注册
	client := &client2.Client{
		Fd:   fd,
		Addr: fmt.Sprintf("%v", sa),
		Buf:  make([]byte, 0, 1024),
		Db:   ctx.GetDb(),
	}
	ctx.RegisterClient(fd, client)
}

// OnRead 处理读事件
func (h *RedisCallbackHandler) OnRead(client *client2.Client, ctx callback.ServerContext) {
	buf := make([]byte, 1024)
	n, err := unix.Read(client.Fd, buf)
	if err != nil {
		// 连接关闭或出错
		fmt.Printf("Connection closed: fd=%d, addr=%s\n", client.Fd, client.Addr)
		unix.Close(client.Fd)
		ctx.RemoveClient(client.Fd)
		return
	}

	// 追加到缓冲区
	client.Buf = append(client.Buf, buf[:n]...)

	// 处理命令
	if bytes.Contains(client.Buf, []byte("\r\n")) {
		cmd := protocol.ParseCommand(client.Buf)
		h.processCommand(client, cmd)
		// 清空缓冲区
		client.Buf = client.Buf[:0]
		// 调度写事件
		ctx.ScheduleWrite(client.Fd)
	}
}

func (h *RedisCallbackHandler) processCommand(client *client2.Client, cmd *types.Command) {
	// 这里需要为 Client 添加Db字段，或者在注册时设置
	switch cmd.Cmd {
	case types.CmdPing:
		client.Reply = protocol.FormatReply("PONG")
	case types.CmdSet:
		if len(cmd.Args) < 2 {
			client.Reply = protocol.FormatError("ERR wrong number of arguments for 'set' command")
			return
		}
		client.Db.Set(cmd.Args[0], cmd.Args[1])
		client.Reply = protocol.FormatSimpleString("OK")
	case types.CmdGet:
		if len(cmd.Args) < 1 {
			client.Reply = protocol.FormatError("ERR wrong number of arguments for 'get' command")
			return
		}
		if value, ok := client.Db.Get(cmd.Args[0]); ok {
			client.Reply = protocol.FormatReply(value)
		} else {
			client.Reply = protocol.FormatReply("")
		}
	case types.CmdDel:
		if len(cmd.Args) < 1 {
			client.Reply = protocol.FormatError("ERR wrong number of arguments for 'del' command")
			return
		}
		client.Db.Delete(cmd.Args[0])
		client.Reply = protocol.FormatSimpleString("OK")
	case types.CmdQuit:
		client.Reply = protocol.FormatSimpleString("BYE")
	default:
		client.Reply = protocol.FormatError("ERR unknown command")
	}
}

func (h *RedisCallbackHandler) OnWrite(client *client2.Client, ctx callback.ServerContext) {
	if len(client.Reply) == 0 {
		return
	}

	n, err := unix.Write(client.Fd, client.Reply)
	if err != nil {
		fmt.Printf("Write err: %v\n", err)
		unix.Close(client.Fd)
		ctx.RemoveClient(client.Fd)
		return
	}

	if n < len(client.Reply) {
		// 未完全写入，保留剩余数据
		client.Reply = client.Reply[n:]

		// 再次调度写事件
		ctx.ScheduleWrite(client.Fd)
	} else {
		// 全部写入完成
		client.Reply = nil
		// 如果是quit命令，关闭连接
		if bytes.HasPrefix(client.Reply, []byte("+BYE")) {
			unix.Close(client.Fd)
			ctx.RemoveClient(client.Fd)
		}
	}
}
