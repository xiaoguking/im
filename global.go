package main

import (
	"encoding/json"
	"flag"
	"fmt"
	_ "fmt"
	"os"
	"sync"
)

type hub struct {
	c map[*connection]bool
	b chan []byte
	r chan *connection
	u chan *connection
}

type us struct {
	m map[string]*connection
}

var (
	h = hub{
		c: make(map[*connection]bool),
		u: make(chan *connection),
		b: make(chan []byte, 512),
		r: make(chan *connection),
	}

	u = us{
		m: make(map[string]*connection),
	}
	//在线的client客户端
	clientList = make(map[string]string)

	//uid 和client 的绑定关系
	uidBindClient = make(map[string][]string)

	//存储uid的离线消息
	uidLogoutMsg = make(map[string][]string)

	//引入锁
	lock sync.Mutex

	//log操作
	logfile *os.File

	//全局配置
	conf *Config
)

//初始化命令行
//初始化log
//初始化配置
func init() {
	var logpath string
	var confpath string

	flag.StringVar(&logpath, "l", "./im.log", "日志文件路径")
	flag.StringVar(&confpath, "c", "./im.conf", "配置文件路径")
	// 解析命令行参数写入注册的flag里
	flag.Parse()

	logFile, err := os.OpenFile(logpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("open log file failed, err:", err)
		return
	}
	logfile = logFile

	myConfig := new(Config)
	myConfig.InitConfig(confpath)
	conf = myConfig

}

func (h *hub) Run() {
	for {
		select {
		case c := <-h.r:
			h.c[c] = true
			c.Data.Ip = c.ws.RemoteAddr().String()
			c.Data.Client = c.client
			c.Data.Type = "0"
			c.Data.UserList = getUserList()
			data, _ := json.Marshal(c.Data)
			c.sc <- data
		case c := <-h.u:
			if _, ok := h.c[c]; ok {
				delete(h.c, c)
				close(c.sc)
			}
		case data := <-h.b:
			for c := range h.c {
				select {
				case c.sc <- data:
				default:
					delete(h.c, c)
					close(c.sc)
				}
			}
		}
	}
}

func LogoutMasRun() {
	for {
		for k, v := range uidLogoutMsg {
			lock.Lock()
			client, ok := uidBindClient[k]
			lock.Unlock()
			if !ok {
				//fmt.Println("没有指定客户端！！")
				continue
			} else {
				//fmt.Println("有指定客户端！！")
				for _, vs := range client {
					//为了并发安全加锁
					lock.Lock()
					c, oks := u.m[vs]
					lock.Unlock()
					if !oks {
						//fmt.Println("没有户端上线！！")
						continue
					} else {
						if len(v) > 0 {
							for _, vm := range v {
								//fmt.Println("发送消息给uid上线的客户端！！")
								c.sc <- []byte(vm)
								continue
							}
						}
					}
				}
				lock.Lock()
				delete(uidLogoutMsg, k)
				lock.Unlock()
			}

		}
	}
}
