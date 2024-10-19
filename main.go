package main

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"

	"github.com/gobwas/ws"
)

func cors(w http.ResponseWriter) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	chatBroadcaster := NewChatBroadcaster()
	chatBroadcaster.Start(runtime.NumCPU())

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
