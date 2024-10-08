package wsmsg

import (
	"encoding/json"
	"log"
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type MsgType int

const (
	None MsgType = iota
	SendMsg
	SendMsgEvent
)

type WsMsg struct {
	Type MsgType `json:"type"`
	Data any     `json:"data"`
}

type SendMsgData struct {
	Text string `json:"text"`
}

func ProcessMessage(msg []byte, conn *net.Conn, conns *[]*net.Conn, chat *[]string) {
	var msgData = json.RawMessage{}
	var jsonMsg = WsMsg{
		Data: &msgData,
	}

	err := json.Unmarshal(msg, &jsonMsg)
	if err != nil {
		log.Printf("Error while parsing websocket message: %s", err.Error())
	}

	if jsonMsg.Type == SendMsg {
		var sendMsgData = SendMsgData{}

		err := json.Unmarshal(msgData, &sendMsgData)
		if err != nil {
			log.Printf("Error while parsing data for websocket message (type = Send): %s", err.Error())
			return
		}

		var eventMsg = WsMsg{
			Type: SendMsgEvent,
			Data: sendMsgData,
		}

		eventMsgBytes, err := json.Marshal(eventMsg)
		if err != nil {
			log.Printf("Error while marshalling message: %s", err.Error())
			return
		}

		for _, conn := range *conns {
			wsutil.WriteServerMessage(*conn, ws.OpText, eventMsgBytes)
		}

		*chat = append(*chat, sendMsgData.Text)
	} else {
		log.Printf("Invalid message from websocket %v", conn)
	}
}
