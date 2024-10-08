package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"slices"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

	"example.com/ws-chat/wsmsg"
)

func wsHandler(w http.ResponseWriter, r *http.Request, conns *[]*net.Conn, chat *[]string) {
	log.Println(r.Method + " " + r.Pattern)

	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		// handle error
		log.Printf("Error while creating websocket: %s", err.Error())
	}

	log.Printf("Created websocket %v\n", &conn)

	*conns = append(*conns, &conn)

	log.Printf("Websockets: %v\n", conns)

	go func() {
		defer func() {
			*conns = slices.DeleteFunc(*conns, func(elem *net.Conn) bool {
				return elem == &conn
			})
			conn.Close()
			log.Printf("%v Closed websocket\n", &conn)
			log.Printf("Websockets: %v\n", conns)
		}()

		for {
			var msgBytes, err = wsutil.ReadClientText(conn)
			if err != nil {
				log.Printf("%v Error: %#v %s", &conn, err, err.Error())

				var _, isClosed = err.(wsutil.ClosedError)
				if isClosed || err.Error() == "EOF" {
					break
				}
			}

			var msg = string(msgBytes)

			log.Printf("%v Recieved %s\n", &conn, msg)
			wsmsg.ProcessMessage(msgBytes, &conn, conns, chat)

			// err = wsutil.WriteServerMessage(conn, ws.OpText, msgBytes)
			// if err != nil {
			// 	log.Printf("%v Error: %#v %s", &conn, err, err.Error())

			// 	var _, isNetOpErr = err.(*net.OpError)
			// 	if isNetOpErr {
			// 		break
			// 	}
			// }
		}
	}()
}

func cors(w http.ResponseWriter) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	
	var chat []string = make([]string, 0)
	var conns []*net.Conn = make([]*net.Conn, 0)

	chat = append(chat, "abc")
	chat = append(chat, "hello")
	chat = append(chat, "world")
	
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

			enc.Encode(chat)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	})

	http.HandleFunc("/chat-ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler(w, r, &conns, &chat)
	})

	http.ListenAndServe(":8080", nil)
}
