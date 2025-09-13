package server

import (
	"fmt"
	"net"

	"github.com/shgang97/redis-go/callbacks"
	client2 "github.com/shgang97/redis-go/client"
	"github.com/shgang97/redis-go/database"

	"golang.org/x/sys/unix"
)

// Server 实现 ServerContext 接口
type Server struct {
	host      string
	port      int
	fd        int
	kq        int
	db        *database.Database
	clients   map[int]*client2.Client
	callbacks callback.EventCallbacks
}

// 确保 Server 实现 ServerContext 接口
var _ callback.ServerContext = (*Server)(nil)

func NewServer(host string, port int, callbacks callback.EventCallbacks) *Server {
	return &Server{
		host:      host,
		port:      port,
		db:        database.NewDatabase(),
		clients:   make(map[int]*client2.Client),
		callbacks: callbacks,
	}
}

func (s *Server) GetKqueue() int {
	return s.kq
}

func (s *Server) RegisterClient(fd int, client *client2.Client) {
	s.clients[fd] = client
}

func (s *Server) RemoveClient(fd int) {
	delete(s.clients, fd)
}

func (s *Server) GetClient(fd int) (*client2.Client, bool) {
	client, ok := s.clients[fd]
	return client, ok
}

func (s *Server) GetDb() *database.Database {
	return s.db
}

func (s *Server) ScheduleWrite(fd int) {
	ev := []unix.Kevent_t{
		{
			Ident:  uint64(fd),
			Filter: unix.EVFILT_WRITE,
			Flags:  unix.EV_ADD | unix.EV_ONESHOT,
			Fflags: 0,
			Data:   0,
			Udata:  nil,
		},
	}
	if _, err := unix.Kevent(s.kq, ev, nil, nil); err != nil {
		fmt.Printf("Kevent error: %v\n", err)
	}
}

// ListenAndServe 启动服务器
func (s *Server) ListenAndServe() error {
	// 创建 Socket
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		return fmt.Errorf("Socket error: %v\n", err)
	}
	s.fd = fd

	// 设置 SO_REUSEADDR
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		unix.Close(fd)
		return fmt.Errorf("setsockopt error: %v\n", err)
	}

	// 绑定地址
	addr := unix.SockaddrInet4{Port: s.port}
	copy(addr.Addr[:], net.ParseIP(s.host).To4())
	if err := unix.Bind(fd, &addr); err != nil {
		unix.Close(fd)
		return fmt.Errorf("bind error: %v\n", err)
	}

	// 监听
	if err := unix.Listen(fd, 1024); err != nil {
		unix.Close(fd)
		return fmt.Errorf("listen error: %v\n", err)
	}

	// 设置非阻塞
	if err := unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return fmt.Errorf("setnonblock error: %v\n", err)
	}

	// 创建 kqueue
	kq, err := unix.Kqueue()
	if err != nil {
		unix.Close(fd)
		return fmt.Errorf("kqueue error: %v\n", err)
	}
	s.kq = kq

	// 注册监听 socket 事件
	change := []unix.Kevent_t{
		{
			Ident:  uint64(fd),
			Filter: unix.EVFILT_READ,
			Flags:  unix.EV_ADD,
			Fflags: 0,
			Data:   0,
			Udata:  nil,
		},
	}
	if _, err := unix.Kevent(s.kq, change, nil, nil); err != nil {
		unix.Close(fd)
		unix.Close(kq)
		return fmt.Errorf("kevent error: %v\n", err)
	}
	fmt.Printf("Server listening on %s:%d\n", s.host, s.port)
	return s.eventLoop()
}

func (s *Server) eventLoop() error {
	events := make([]unix.Kevent_t, 64)

	for {
		// 等待事件
		n, err := unix.Kevent(s.kq, nil, events, nil)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return fmt.Errorf("kevent wait error: %v\n", err)
		}

		// 处理事件
		for i := 0; i < n; i++ {
			ev := events[i]
			ident := int(ev.Ident)

			// 错误事件
			if ev.Flags&unix.EV_ERROR != 0 {
				fmt.Printf("Kevent error: fd=%d\n", ident)
				if client, ok := s.clients[int(ident)]; ok {
					unix.Close(client.Fd)
					delete(s.clients, ident)
				}
				continue
			}

			// 监听 socket 事件（新连接）
			if ident == s.fd {
				nfd, sa, err := unix.Accept(s.fd)
				if err != nil {
					fmt.Printf("Accept error: %v\n", err)
					continue
				}
				s.callbacks.OnAccept(nfd, sa, s)
				continue
			}

			// 客户端事件
			//time.Sleep(time.Second * 3600)
			if client, ok := s.clients[ident]; ok {
				switch ev.Filter {
				case unix.EVFILT_READ:
					s.callbacks.OnRead(client, s)

				case unix.EVFILT_WRITE:
					s.callbacks.OnWrite(client, s)
				}
			}
		}
	}
}

func (s *Server) Close() {
	for _, client := range s.clients {
		unix.Close(client.Fd)
	}
	unix.Close(s.fd)
	unix.Close(s.kq)
}
