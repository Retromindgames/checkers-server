package message

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type Message struct {
	Command string  `json:"command"`
	Value   float64 `json:"value"`
}

func ParseMessage(msg []byte, conn *websocket.Conn) (*Message, error) {
	var message Message
	if err := json.Unmarshal(msg, &message); err != nil {
		return nil, fmt.Errorf("invalid message format: %v", err)
	}
	return &message, nil
}
