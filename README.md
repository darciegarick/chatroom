#### 使用 Golang 实现一个 TCP 聊天室 

1. 新建一个 `chatroom` 文件夹项目

```shell
$ mkdir -p chatroom
$ cd chatroom
$ go mod init chatroom
```

2. 新建服务端代码文件：`server.go`

```shell
$ vim server.go
```
一步步的实现，定义`User`结构体和入口`main`函数
```go
// chatroom/server.go
package main

type User struct {
    ID             int         // ID 是用户唯一标识，通过 GenUserID 函数生成；
    Addr           string      // Addr 是用户的 IP 地址和端口；
    EnterAt        time.Time   // EnterAt 是用户进入时间；
    MessageChannel chan string // MessageChannel 是当前用户发送消息的通道；
}

// 定义一个 idCounter，保护 id 唯一
var (
    nextId    int
    idCounter sync.Mutex
)

var (
    // 新用户到来，通过该 channel 进行登记 (用户进入聊天室时，发送通知)
    enteringChannel = make(chan *User)
    // 用户离开，通过该 channel 进行登记 (用户离开聊天室时发送通知)
    leavingChannel = make(chan *User)
    // 广播专用的用户普通消息 channel，缓冲是尽可能避免出现异常情况堵塞 
    messageChannel = make(chan string, 8)
)

func main() {
  // 只绑定在 127.0.0.1 上：net.Listen(“tcp”, “127.0.0.1:2020”)
  // 如果不指定 IP 会绑定到当前机器所有的 IP 上
    listener, err := net.Listen("tcp", "127.0.0.1:2020")
  // 同一个网络环境，如果要别的设备可访问的话，可以将 ip、端口设置为：0.0.0.0:2020
    if err != nil {
        panic(err)
    }

    go broadcaster()

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Panicln(err)
            continue
        }

        go handleConn(conn)
    }
}

// handleConn 函数用于 处理 客户端连接的函数 handleConn
func handleConn(conn net.Conn) {
}

// broadcaster 函数用于记录聊天用户，并进行消息广播
func broadcaster() {
}

// genUserID 用于生成用户ID的
func genUserID() int {
    idCounter.Lock()
    defer idCounter.Unlock()

    nextId++
    return nextId
}

// sendMessage 函数：用户发送消息
func sendMessage(conn net.Conn, ch <-chan string) {
}
```
上面就是整体的架构

3. 实现服务端的`handleConn`函数
  - 利用`User`构建新用户，生成新用户的实例
  ```go
  user := &User{
      ID:             genUserID(),
      Addr:           conn.RemoteAddr().String(),
      EnterAt:        time.Now(),
      MessageChannel: make(chan string, 8),
  }
  ```

  - 新开一个 `goroutine` 给用户发消息
  ```go
  go sendMessage(conn, user.MessageChannel)

  // 除此之外还有单向的 channel：只能接收（<-chan，only receive）和只能发送（chan<-， only send）。
  // 它们没法直接创建，而是通过正常（双向）channel 转换而来（会自动隐式转换）。
  // 它们存在的价值，主要是避免 channel 被乱用。上面代码中 ch <-chan string 就是为了限制在 sendMessage 函数中只从 channel 读数据，不允许往里写数据。
  func sendMessage(conn net.Conn, ch <-chan string) {
      for msg := range ch {
          fmt.Fprintln(conn, msg)
      }
  }
  ```

  - 
