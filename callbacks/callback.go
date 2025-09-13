package callback

import (
	"github.com/shgang97/redis-go/client"
	"github.com/shgang97/redis-go/database"
	"golang.org/x/sys/unix"
)

// ServerContext 定义服务器需要提供给处理器的功能
type ServerContext interface {
	GetKqueue() int
	RegisterClient(fd int, client *client.Client)
	RemoveClient(fd int)
	GetClient(fd int) (*client.Client, bool)
	GetDb() *database.Database
	ScheduleWrite(fd int)
}

// EventCallbacks 定义事件回调接口
type EventCallbacks interface {
	OnAccept(fd int, sa unix.Sockaddr, ctx ServerContext)
	OnRead(client *client.Client, ctx ServerContext)
	OnWrite(client *client.Client, ctx ServerContext)
}
