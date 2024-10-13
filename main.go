package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

	"ws-chat/wsmsg"
)

func HandleWsMsg(msg []byte, chatBroadcastChan chan <- wsmsg.SendMsgData) {
	wsMsg, err := wsmsg.ParseMessageType(msg)
	if err != nil {
		log.Printf("Error while parsing websocket message: %s", err.Error())
	}

	if wsMsg.Type == wsmsg.SendMsg {
		var msgData wsmsg.SendMsgData

		err = json.Unmarshal(wsMsg.Data, &msgData)
		if err != nil {
			log.Printf("Error while parsing data for websocket message (type = %s): %s", wsMsg.Type, err.Error())
			return
		}

		log.Println("Sending to broadcaster")

		chatBroadcastChan <- msgData
	}

}

func wsHandler(w http.ResponseWriter, r *http.Request, broadcaster *ChatBroadcaster) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		// handle error
		log.Printf("Error while creating websocket: %s", err.Error())
	}

	log.Printf("Created websocket %v\n", &conn)

	wsconn := WsConn{
		conn:     &conn,
		sendChan: make(chan wsmsg.SendMsgData, 10),
	}
	broadcaster.ConnChan <- wsconn

	go func() {
		defer func() {
			close(wsconn.sendChan)
			(*wsconn.conn).Close()
		}()


		for {
			log.Printf("Ws %p: Loop", wsconn.conn)

			select {
			case chatMsg := <- wsconn.sendChan:
				log.Printf("%p Recieved chat msg event", wsconn.conn)

				eventMsg := wsmsg.WsMsg{
					Type: wsmsg.SendMsgEvent,
					Data: chatMsg,
				}

				eventMsgBytes, err := json.Marshal(eventMsg)
				if err != nil {
					log.Printf("Error while marshalling message: %s", err.Error())
					break
				}
				
				err = wsutil.WriteServerText(*wsconn.conn, eventMsgBytes)
				if err != nil {
					log.Printf("%p Error: %s", wsconn.conn, err.Error())
				}


			default:
				var msg, err = wsutil.ReadClientText(conn)
				if err != nil {
					log.Printf("%v Error: %#v %s", &conn, err, err.Error())

					var _, isClosed = err.(wsutil.ClosedError)
					if isClosed || err.Error() == "EOF" {
						return
					}
				}

				log.Printf("%v Recieved %s\n", &conn, string(msg))

				HandleWsMsg(msg, broadcaster.ChatChan)

				// err = wsutil.WriteServerMessage(conn, ws.OpText, msgBytes)
				// if err != nil {
				// 	log.Printf("%v Error: %#v %s", &conn, err, err.Error())

				// 	var _, isNetOpErr = err.(*net.OpError)
				// 	if isNetOpErr {
				// 		break
				// 	}
				// }
			}
		}
	}()
}

func cors(w http.ResponseWriter) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
}

type WsConn struct {
	conn     *net.Conn
	sendChan chan wsmsg.SendMsgData
}

type ChatBroadcaster struct {
	ChatChan chan wsmsg.SendMsgData
	ConnChan chan WsConn

	conns []WsConn
}

func NewChatBroadcaster() ChatBroadcaster {
	var chat = make(chan wsmsg.SendMsgData, 10)
	var newConns = make(chan WsConn, 10)

	var conns = make([]WsConn, 0)

	return ChatBroadcaster{
		ChatChan: chat,
		ConnChan: newConns,
		conns:    conns,
	}
}

func (broadcaster *ChatBroadcaster) Start() {
	defer log.Println("Broadcaster: Exiting")

	for {
		log.Println("Broadcaster: Loop")

		select {
		case msg := <- broadcaster.ChatChan:
			log.Printf("Broadcaster: received Chat msg %#v", msg)
			for _, conn := range broadcaster.conns {
				log.Printf("Broadcaster: sending Chat msg %#v to %p", msg, conn.conn)
				conn.sendChan <- msg
			}

		case conn := <-broadcaster.ConnChan:
			log.Printf("Broadcaster: New connection")
			broadcaster.conns = append(broadcaster.conns, conn)
			log.Printf("Broadcaster: Active connections %v", broadcaster.conns)
		}
	}
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	var chatBroadcaster = NewChatBroadcaster()
	go chatBroadcaster.Start()

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
			wsHandler(w, r, &chatBroadcaster)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	http.ListenAndServe(":8080", nil)
}
