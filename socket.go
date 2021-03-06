package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

type Msg struct {
	Cmd    int    `json:"cmd"`    //消息指令
	Body   Body   `json:"body"`   //消息实体
	Client string `json:"client"` //指定的客户端标识码
	Uid    string `json:"uid"`    //指定的uid
	Group  string `json:"group"`  //群组id
}

type Body struct {
	Type      int    `json:"type"`      //消息类型
	User      string `json:"user"`      //发送者的uid
	Content   string `json:"content"`   //消息文本内容消息
	Time      string `json:"time"`      //发送时间
	Extension string `json:"extension"` //扩展数据
	Image     string `json:"image"`     //图片消息
}

func process(conn net.Conn) {
	defer conn.Close() // 关闭连接
	msg := &Msg{}
	for {
		reader := bufio.NewReader(conn)
		var buf [512]byte
		n, err := reader.Read(buf[:]) // 读取数据
		if err != nil {
			ErrorLogs(fmt.Sprintf("socket 客户端连接异常 err %v", err))
			break
		}
		recvStr := string(buf[:n])
		SuccessLogs(fmt.Sprintf("收到socket客户端消息 %v", recvStr))
		json.Unmarshal(buf[:n], &msg)
		m, _ := json.Marshal(msg.Body)
		switch {
		case msg.Cmd == CMD_SEND_TO_ALL: //发送全局广播消息
			h.b <- m
			w, errs := conn.Write([]byte(Success(""))) // 发送数据
			if errs != nil {
				ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
				break
			}
			SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
		case msg.Cmd == CMD_CLIENT_SEND_TO_ONE: //给指定客户端发消息
			if _, ok := u.m[msg.Client]; ok { //判断客户端是否在线
				client := u.m[msg.Client]
				client.sc <- m
				w, errs := conn.Write([]byte(Success(""))) // 发送数据
				if errs != nil {
					ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
					break
				}
				SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
			} else {
				w, errs := conn.Write([]byte(Error("指定客户端不在线")))
				if errs != nil {
					ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
					break
				}
				SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
			}
		case msg.Cmd == CMD_GET_ALL_CLIENT: //获取在线的client客户端
			w, errs := conn.Write([]byte(Success(clientList)))
			if errs != nil {
				ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
				break
			}
			SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
		case msg.Cmd == CMD_BIND_UID: //将uid绑定到指定客户端上
			clientId := msg.Client
			Uid := msg.Uid
			if clientId != "" && Uid != "" {
				key := Uid
				value, ok := uidBindClient[key]
				if !ok {
					value = make([]string, 0, 1)
				}
				value = append(value, clientId)
				uidBindClient[key] = value
				w, errs := conn.Write([]byte(Success(""))) // 发送数据
				if errs != nil {
					ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
					break
				}
				SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
			}
		case msg.Cmd == CMD_SEND_TO_UID: //向指定uid发消息
			uid := msg.Uid
			lock.Lock()
			client_list, ok := uidBindClient[uid] //uid 绑定的client
			lock.Unlock()
			if !ok {
				//没有在线的客户端 将消息存为离线消息
				lock.Lock()
				value, ok := uidLogoutMsg[uid]
				lock.Unlock()
				if !ok {
					value = make([]string, 0, 1)
				}
				value = append(value, string(m))
				lock.Lock()
				uidLogoutMsg[uid] = value
				lock.Unlock()
				//fmt.Println(uidLogoutMsg)
				w, errs := conn.Write([]byte(Error("发送离线消息成功")))
				if errs != nil {
					ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
					break
				}
				SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
				break
			}
			//fmt.Println("uid 绑定客户端的:", client_list)
			for _, v := range client_list {
				client, ok := clientList[v]
				if !ok {
					w, errs := conn.Write([]byte(Error("uid 没有在线的客户端"))) // 发送数据
					if errs != nil {
						ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
					}
					SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
					continue
				}
				if _, ok := u.m[client]; ok { //判断客户端是否在线
					client := u.m[v]
					client.sc <- m
					w, errs := conn.Write([]byte(Success(""))) // 发送数据
					if errs != nil {
						ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
						break
					}
					SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
				}
				w, errs := conn.Write([]byte(Error("uid 没有在线的客户端"))) // 发送数据
				if errs != nil {
					ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
					break
				}
				SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
			}
		case msg.Cmd == CMD_GET_CLIENT_ID_BY_UID: //获取在线uid的客户端
			w, errs := conn.Write([]byte(Success(uidBindClient[msg.Uid]))) // 发送数据
			if errs != nil {
				ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
				break
			}
			SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
		case msg.Cmd == CMD_JOIN_GROUP: //加入群组

		case msg.Cmd == CMD_KICK: //踢出某一个客户端
			client := msg.Client
			c, ok := u.m[client]
			if !ok {
				w, errs := conn.Write([]byte(Success("客户端不在线"))) // 发送数据
				if errs != nil {
					ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
					break
				}
				SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
				break
			}
			c.ws.Close()                                       //断开链接
			w, errs := conn.Write([]byte(Success(clientList))) // 发送数据
			if errs != nil {
				ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
				break
			}
			SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
		default:
			w, errs := conn.Write([]byte("消息类型错误")) // 发送数据
			if errs != nil {
				ErrorLogs(fmt.Sprintf("socket 客户端消息发送失败 error: %v", errs))
				break
			}
			SuccessLogs(fmt.Sprintf("发送成功 字节数 %v", w))
		}

	}
}
func SocketRun() {
	listen, err := net.Listen("tcp", "0.0.0.0:12356")

	if err != nil {
		fmt.Println("listen failed, err:", err)
		return
	}
	fmt.Println("socket start " + " tcp://0.0.0.0:12356")
	fmt.Println("\r" +
		"  _ _ _     ___  ____ \n " +
		"  / /    / __ `__/ / \n" +
		"  / /    / / / / / /  \n" +
		"_/_/_   /_/ /_/ /_/  ")
	fmt.Printf("\nIM version %v  %v", conf.Read("default", "version"), "2021-9-10 15:04:05")
	for {
		conn, err := listen.Accept() // 建立连接
		if err != nil {
			fmt.Println("连接异常")
			continue
		}
		go process(conn) // 启动一个goroutine处理连接
	}
}
