package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"ws-chat/wsmsg"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type ChatWsConn struct {
	conn     *net.Conn
	SendChan chan<- wsmsg.SendMsgData // sending to this channel will send a SendMsgEvent to the websocket
	RecvChan <-chan wsmsg.SendMsgData // this channel emits recieved SendMsg messages from the websocket
}

func StartChatRecvChan(conn *net.Conn, recvChan chan wsmsg.SendMsgData, done <-chan bool) <-chan error {
	// defer (*chatws.conn).Close()
	// defer close(sendChan)
	// defer close(recvChan)

	log.Printf("Ws [%p]: Starting Recieve goroutine", conn)

	errChan := make(chan error)

	go func() {
		for {
			select {
			case <-done:
				return

			default:
				header, err := ws.ReadHeader(*conn)
				if err != nil {
					log.Printf("Ws [%p] Reciever: Error %s, closing connection", conn, err.Error())
					errChan <- err
					return
				}

				if header.OpCode == ws.OpClose {
					log.Printf("Ws [%p] Reciever: Recieved close opcode, closing connection", conn)
					errChan <- nil
					return
				} else if header.OpCode != ws.OpText {
					continue
				}

				msg := make([]byte, header.Length)
				_, err = io.ReadFull(*conn, msg)

				if err != nil {
					log.Printf("Ws [%p] Reciever: Error %s, closing connection", conn, err.Error())
					errChan <- err
					return
				}

				if header.Masked {
					ws.Cipher(msg, header.Mask, 0)
				}

				log.Printf("Ws [%p] Reciever: Recieved msg [%s]", conn, string(msg))

				wsMsg, err := wsmsg.ParseMessageType(msg)
				if err != nil {
					log.Printf("Ws [%p] Reciever: Error while parsing websocket message: %s", conn, err.Error())
				}

				if wsMsg.Type == wsmsg.SendMsg {
					var msgData wsmsg.SendMsgData

					err = json.Unmarshal(wsMsg.Data, &msgData)
					if err != nil {
						log.Printf("Ws [%p] Reciever: Error while parsing data for websocket message (type = %s): %s", conn, wsMsg.Type, err.Error())
					}

					log.Printf("Ws [%p] Reciever: Sending %#v to recv channel", conn, msgData)

					recvChan <- msgData
				}
			}
		}
	}()

	return errChan
}

func StartChatSendChan(conn *net.Conn, sendChan chan wsmsg.SendMsgData, done <-chan bool) <-chan error {
	log.Printf("Ws [%p]: Starting Send goroutine", conn)

	errChan := make(chan error)

	go func() {
		for {
			select {
			case <-done:
				return

			case chatMsg := <-sendChan:

				log.Printf("Ws [%p] Sender: Recieved chat msg event", conn)

				eventMsg := wsmsg.WsMsg{
					Type: wsmsg.SendMsgEvent,
					Data: chatMsg,
				}

				eventMsgBytes, err := json.Marshal(eventMsg)
				if err != nil {
					log.Printf("Ws [%p] Sender: Error while marshalling message: %s", conn, err.Error())
				}

				err = wsutil.WriteServerText(*conn, eventMsgBytes)
				if err != nil {
					log.Printf("Ws [%p] Sender: Error: %s", conn, err.Error())
				}
			}
		}
	}()

	return errChan
}

func NewChatWsConn(conn *net.Conn) ChatWsConn {
	sendChan := make(chan wsmsg.SendMsgData, 10)
	recvChan := make(chan wsmsg.SendMsgData, 10)

	wsconn := ChatWsConn{
		conn:     conn,
		SendChan: sendChan,
		RecvChan: recvChan,
	}

	// done := make(chan bool)
	recvDone := make(chan bool)
	sendDone := make(chan bool)

	recvErrChan := StartChatRecvChan(conn, recvChan, recvDone)
	sendErrChan := StartChatSendChan(conn, sendChan, sendDone)

	go func() {
		select {
		case err := <-recvErrChan:
			log.Printf("Ws [%p]: Recieved error [%s] from reciever chan, closing websocket", conn, err)
			sendDone <- true

		case err := <-sendErrChan:
			log.Printf("Ws [%p]: Recieved error [%s] from sender chan, closing websocket", conn, err)
			recvDone <- true
		}

		log.Printf("Closing chat ws [%p] and its send/recieve channels", conn)
		(*conn).Close()
		close(sendChan)
		close(recvChan)
	}()

	return wsconn
}
