package wsmsg

import (
	"encoding/json"
)

type MsgType int

const (
	None MsgType = iota
	SendMsg
	SendMsgEvent
)

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

type MsgData interface{}

type WsMsg struct {
	Type MsgType `json:"type"`
	Data MsgData `json:"data,omitempty"`
}

type WsMsgRaw struct {
	Type MsgType         `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type SendMsgData struct {
	Text string `json:"text"`
}

func ParseMessageType(msg []byte) (WsMsgRaw, error) {
	var wsMsg = WsMsgRaw{}

	var err = json.Unmarshal(msg, &wsMsg)
	if err != nil {
		return WsMsgRaw{}, err
	}

	return wsMsg, nil
}
