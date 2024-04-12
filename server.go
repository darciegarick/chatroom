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
	// 新用户到来，通过该 channel 进行登记
	enteringChannel = make(chan *User)
	// 用户离开，通过该 channel 进行登记
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
