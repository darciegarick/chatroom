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

  - 给用户发送信息
  ```go
  user.MessageChannel <- "欢迎： " + strconv.Itoa(user.ID)

  messageChannel <- "user: `" + strconv.Itoa(user.ID) + "` has enter"
  ```

  - 将当前用户加入聊天室列表，通过`channel`来写入，避免了锁
  ```go
  enteringChannel <- user
  ```

  - 循环读取用户输入的内容
  ```go
  input := bufio.NewScanner(conn)
  for input.Scan() {
      messageChannel <- strconv.Itoa(user.ID) + ": " + input.Text()
  }

  if err := input.Err(); err != nil {
      log.Println("读取错误：", err)
  }
  ```

  - 用户离开
  ```go
  leavingChannel <- user
  messageChannel <- "user: `" + strconv.Itoa(user.ID) + "` has left"
  ```

  实现`broadcaster`函数，用于记录用户行为，广播通知聊天室用户
  ```go
  func broadcaster() {
        users := make(map[*User]struct{})

        for {
            select {
            case user := <-enteringChannel:
                // 新用户进入
                users[user] = struct{}{}
            case user := <-leavingChannel:
                // 用户离开
                delete(users, user)
                // 避免 goroutine 泄露
                close(user.MessageChannel)
            case msg := <-messageChannel:
                // 给所有在线用户发送消息
                for user := range users {
                    user.MessageChannel <- msg
                }
            }
        }
    }
  ```

4. 新建客户端代码文件：`client.go`
由于都是属于 main 包，所以可以将`clien.go`放在`chatroom/client/client.go`
```go
// chatroom/client/client.go
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
```
上面就是客户端的完整代码。

下面提供服务端的完整代码：
```go
package main

import (
    "bufio"
    "fmt"
    "log"
    "net"
    "strconv"
    "sync"
    "time"
)

type User struct {
    ID             int
    Addr           string
    EnterAt        time.Time
    MessageChannel chan string
}

// 定义一个 idCounter，用户保护 id 唯一
var (
    nextId    int
    idCounter sync.Mutex
)

var (
    // 新用户到来，通过该 channel 进行登记
    enteringChannel = make(chan *User)
    // 用户离开，通过该 channel 进行登记
    leavingChannel = make(chan *User)
    // 广播专用的用户普通消息 channel，缓冲是尽可能避免出现异常情况堵塞，这里简单给了 8，具体值根据情况调整
    messageChannel = make(chan string, 8)
)

func main() {
    listener, err := net.Listen("tcp", "127.0.0.1:2020")
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

// broadcaster 用于记录聊天室用户，并进行消息广播：
// 1. 新用户进来；2. 用户普通消息；3. 用户离开
// 这里关键有 3 点：
// 负责登记/注销用户，通过 map 存储在线用户；
// 用户登记、注销，使用专门的 channel。在注销时，除了从 map 中删除用户，还将 user 的 MessageChannel 关闭，避免上文提到的 goroutine 泄露问题；
// 全局的 messageChannel 用来给聊天室所有用户广播消息；
func broadcaster() {
    users := make(map[*User]struct{})

    for {
        select {
        case user := <-enteringChannel:
            // 新用户进入
            users[user] = struct{}{}
        case user := <-leavingChannel:
            // 用户离开
            delete(users, user)
            // 避免 goroutine 泄露
            close(user.MessageChannel)
        case msg := <-messageChannel:
            // 给所有在线用户发送消息
            for user := range users {
                user.MessageChannel <- msg
            }
        }
    }
}

func handleConn(conn net.Conn) {
    defer conn.Close()

    // 1. 新用户进来，构建该用户的实例
    user := &User{
        ID:             genUserID(),
        Addr:           conn.RemoteAddr().String(),
        EnterAt:        time.Now(),
        MessageChannel: make(chan string, 8),
    }

    // 2. 当前在一个新的 goroutine 中，用来进行读操作，因此需要开一个 goroutine 用于写操作
    // 读写 goroutine 之间可以通过 channel 进行通信
    go sendMessage(conn, user.MessageChannel)

    // 3. 给当前用户发送欢迎信息
    user.MessageChannel <- "欢迎你的到来：" + strconv.Itoa(user.ID)
    // 知识点
    // string 转成 int：
    // int, err := strconv.Atoi(string)
    // string 转成 int64：
    // int64, err := strconv.ParseInt(string, 10, 64)
    // int 转成 string：
    // string := strconv.Itoa(int)
    // int64 转成 string：
    // string := strconv.FormatInt(int64,10)

    // 同时给聊天室所有用户发送有新用户到来的提醒；
    messageChannel <- "user:`" + strconv.Itoa(user.ID) + "` has enter"

    // 4. 将该记录到全局的用户列表中，避免用锁
    // 注意，这里和 3）的顺序不能反，否则自己会收到自己到来的消息提醒；（当然，我们也可以做消息过滤处理）
    enteringChannel <- user

    // 5. 循环读取用户的输入
    input := bufio.NewScanner(conn)
    for input.Scan() {
        messageChannel <- strconv.Itoa(user.ID) + ": " + input.Text()
    }

    if err := input.Err(); err != nil {
        log.Println("读取错误：", err)
    }

    // 6. 用户离开
    leavingChannel <- user
    messageChannel <- "user:`" + strconv.Itoa(user.ID) + "` has left"
}

func genUserID() int {
    idCounter.Lock()
    defer idCounter.Unlock()

    nextId++
    return nextId
}

// channel 实际上有三种类型，大部分时候，我们只用了其中一种，就是正常的既能发送也能接收的 channel。
// 除此之外还有单向的 channel：只能接收（<-chan，only receive）和只能发送（chan<-， only send）。
// 它们没法直接创建，而是通过正常（双向）channel 转换而来（会自动隐式转换）。
// 它们存在的价值，主要是避免 channel 被乱用。上面代码中 ch <-chan string 就是为了限制在 sendMessage 函数中只从 channel 读数据，不允许往里写数据。
func sendMessage(conn net.Conn, ch <-chan string) {
    for msg := range ch {
        fmt.Fprintln(conn, msg)
    }
}
```

更新，Golang 原生 os 库终端输入支持不太好，使用`github.com/c-bata/go-prompt`优化一波。
首先导入：
```go
import (
    "bufio"
    "fmt"
    "log"
    "net"
    "os"
    "strings"

    "github.com/c-bata/go-prompt"
)
```

连接端口方式不变：
```go
func main() {
    conn, err := net.Dial("tcp", "127.0.0.1:2020")
    if err != nil {
        panic(err)
    }

  // 创建一个类型为 struct{} 的通道 done，用于在主 goroutine 和后台 goroutine 之间进行同步。
  done := make(chan struct{})

  // ...
 }
```

删除`mustCopy`函数，使用新引入的包进行优化输入输出：
```go
func main() {
  //...

    // 接收消息
    go func() {
        scanner := bufio.NewScanner(conn)
        for scanner.Scan() {
            fmt.Printf("\r%s\n>>> ", scanner.Text())
        }
        if scanner.Err() != nil {
            log.Fatalf("Failed to read from server: %v", scanner.Err())
        }
        done <- struct{}{}
    }()

    // 发送消息
    go func() {
        p := prompt.New(
            func(in string) {
                in = strings.TrimSpace(in)
                if in == "quit" || in == "exit" {
                    fmt.Println("退出聊天室...")
                    conn.Close()
                    os.Exit(0)
                    return
                }
                if in == "" {
                    return
                }
                _, err := conn.Write([]byte(in + "\n"))
                if err != nil {
                    log.Fatalf("Failed to write to server: %v", err)
                }
            },
            func(d prompt.Document) []prompt.Suggest {
                // TODO 根据情况添加自动完成的建议
                return []prompt.Suggest{}
            },
            prompt.OptionPrefix(">>> "),
            prompt.OptionPrefixTextColor(prompt.Yellow),
        )
        p.Run()
    }()
  
  <-done
}
```