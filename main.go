package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	handler2 "github.com/shgang97/redis-go/handler"
	server2 "github.com/shgang97/redis-go/server"
)

func main() {
	host := flag.String("h", "127.0.0.1", "Server host")
	port := flag.Int("p", 6379, "Server port")

	// 创建回调处理器
	handler := &handler2.RedisCallbackHandler{}

	// 创建服务器
	server := server2.NewServer(*host, *port, handler)

	// 处理中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down server...")
		server.Close()
		os.Exit(1)
	}()

	// 启动服务器
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}
