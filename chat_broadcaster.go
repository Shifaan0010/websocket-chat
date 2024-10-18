package main

import (
	"log"
	"slices"
	"ws-chat/wsmsg"
)

type ChatBroadcaster struct {
	ChatChan chan wsmsg.SendMsgData
	ConnChan chan ChatWsConn

	conns []ChatWsConn
}

func NewChatBroadcaster() ChatBroadcaster {
	var chat = make(chan wsmsg.SendMsgData, 10)
	var newConns = make(chan ChatWsConn, 10)

	var conns = make([]ChatWsConn, 0)

	return ChatBroadcaster{
		ChatChan: chat,
		ConnChan: newConns,
		conns:    conns,
	}
}

func (broadcaster *ChatBroadcaster) Start(goroutineCount int) {
	for i := 0; i < goroutineCount; i += 1 {
		go func() {
			defer log.Printf("Broadcaster [%d]: Exiting", i)

			for {
				log.Printf("Broadcaster [%d]: Loop", i)

				select {
				case msg := <-broadcaster.ChatChan:
					log.Printf("Broadcaster [%d]: received Chat msg %#v", i, msg)
					for _, conn := range broadcaster.conns {
						log.Printf("Broadcaster [%d]: sending Chat msg %#v to %p", i, msg, conn.conn)
						conn.SendChan <- msg
					}

				case conn := <-broadcaster.ConnChan:
					log.Printf("Broadcaster [%d]: New connection", i)
					broadcaster.conns = append(broadcaster.conns, conn)
					go func() {
						for chatMsg := range conn.RecvChan {
							broadcaster.ChatChan <- chatMsg
						}

						log.Printf("Broadcaster [%d]: recv channel closed for ws [%p], removing websocket from active connections", i, conn)
						broadcaster.conns = slices.DeleteFunc(broadcaster.conns, func(elem ChatWsConn) bool {
							return elem.conn == conn.conn
						})
						log.Printf("Broadcaster [%d]: Active connections %v", i, broadcaster.conns)
					}()
					log.Printf("Broadcaster [%d]: Active connections %v", i, broadcaster.conns)
				}
			}
		}()
	}
}
