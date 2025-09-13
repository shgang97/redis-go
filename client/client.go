package client

import "github.com/shgang97/redis-go/database"

// Client 客户端连接结构
type Client struct {
	Fd    int
	Addr  string
	Buf   []byte
	Reply []byte
	Db    *database.Database
}
