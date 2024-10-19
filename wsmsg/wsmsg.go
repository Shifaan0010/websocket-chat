package wsmsg

import (
	"encoding/json"
	"log"
)

type MsgType int

type MsgData interface{}

type WsMsg struct {
	Type MsgType `json:"type"`
	Data MsgData `json:"data,omitempty"`
}

type wsMsgRaw struct {
	Type MsgType         `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

const (
	None MsgType = iota
	SendMsg
	SendMsgEvent
)

type SendMsgData struct {
	Text string `json:"text"`
}

func (msgType MsgType) String() string {
	if msgType == None {
		return "None"
	} else if msgType == SendMsg {
		return "SendMsg"
	} else if msgType == SendMsgEvent {
		return "SendMsgEvent"
	} else {
		return "Unknown"
	}
}

func parseMessageType(msg []byte) (wsMsgRaw, error) {
	var wsMsg = wsMsgRaw{}

	var err = json.Unmarshal(msg, &wsMsg)
	if err != nil {
		return wsMsgRaw{}, err
	}

	return wsMsg, nil
}

func ParseMessage(msg []byte) (MsgType, MsgData, error) {
	msgRaw, err := parseMessageType(msg)
	if err != nil {
		log.Printf("Error while parsing websocket message: %s", err.Error())
		return None, nil, err
	}

	msgType := msgRaw.Type

	var msgData MsgData = nil

	if msgType == SendMsg {
		var sendMsgData SendMsgData

		err = json.Unmarshal(msgRaw.Data, &sendMsgData)
		if err != nil {
			log.Printf("Error while parsing data for websocket message (type = %s): %s", msgType, err.Error())
			return None, nil, err
		}

		msgData = sendMsgData
	}

	return msgType, msgData, nil
}

func EncodeMessage(msg WsMsg) ([]byte, error) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error while marshalling message: %s", err.Error())
		return nil, err
	}

	return msgBytes, nil
}
