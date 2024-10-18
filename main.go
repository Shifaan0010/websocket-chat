package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"slices"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

	"ws-chat/wsmsg"
)

// func HandleWsMsg(msg []byte, chatBroadcastChan chan<- wsmsg.SendMsgData) {
// 	wsMsg, err := wsmsg.ParseMessageType(msg)
// 	if err != nil {
// 		log.Printf("Error while parsing websocket message: %s", err.Error())
// 	}
//
// 	if wsMsg.Type == wsmsg.SendMsg {
// 		var msgData wsmsg.SendMsgData
//
// 		err = json.Unmarshal(wsMsg.Data, &msgData)
// 		if err != nil {
// 			log.Printf("Error while parsing data for websocket message (type = %s): %s", wsMsg.Type, err.Error())
// 			return
// 		}
//
// 		log.Printf("Sending %#v to broadcaster", msgData)
//
// 		chatBroadcastChan <- msgData
// 	}
//
// }

func wsHandler(w http.ResponseWriter, r *http.Request, broadcaster *ChatBroadcaster) {
}

func cors(w http.ResponseWriter) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
}

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

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	chatBroadcaster := NewChatBroadcaster()
	go chatBroadcaster.Start(runtime.NumCPU())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method + " " + r.Pattern)

		cors(w)

		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method + " " + r.Pattern)

		cors(w)

		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)

			var enc = json.NewEncoder(w)

			enc.Encode([]string{})
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	})

	http.HandleFunc("/chat-ws", func(w http.ResponseWriter, r *http.Request) {
		log.Printf(r.Method + " " + r.Pattern)

		if r.Method == http.MethodGet {
			conn, _, _, err := ws.UpgradeHTTP(r, w)
			if err != nil {
				// handle error
				log.Printf("Error while creating websocket: %s", err.Error())
			}

			log.Printf("Created websocket %p\n", &conn)
			wsconn := NewChatWsConn(&conn)

			chatBroadcaster.ConnChan <- wsconn
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	http.ListenAndServe(":8080", nil)
}
