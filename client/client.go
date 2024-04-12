package main

import (
	"io"
	"log"
	"net"
	"os"
)

func main() {
	// 建立上面服务端启动好的 IP 和端口连接
	// net.Dial 是一个用于建立网络连接的函数。
	// "tcp" 是网络参数，指定要建立的连接是基于 TCP 协议的。
	// "127.0.0.1:2020" 是地址参数，表示要连接的目标主机和端口。127.0.0.1: 表示本地主机，而 2020 是目标端口号。
	conn, err := net.Dial("tcp", "127.0.0.1:2020")
	if err != nil {
		panic(err)
	}

	// 创建一个类型为 struct{} 的通道 done，用于在主 goroutine 和后台 goroutine 之间进行同步。
	done := make(chan struct{})

	// 启动一个后台 goroutine，该 goroutine 使用 io.Copy 将 conn（一个网络连接）的内容复制到标准输出（os.Stdout）。
	// 注意，这里忽略了错误处理。在复制完成后，输出 "done" 到日志中，并通过 done 通道发送一个空结构体的值，以向主 goroutine 发送一个信号。
	go func() {
		io.Copy(os.Stdout, conn) // NOTE: ignoring errors
		log.Println("done")
		done <- struct{}{} // signal the main goroutine
	}()

	// 调用函数 mustCopy，将标准输入（os.Stdin）的内容复制到 conn（网络连接）中。
	mustCopy(conn, os.Stdin)
	conn.Close()
	<-done
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}
