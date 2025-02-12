package models

import "github.com/gorilla/websocket"

type Player struct {
	ID        string          `json:"id"`
	Token     string          `json:"token"`
	SessionID string          `json:"sessionid"`
	Currency  string          `json:"currency"`
	CurrencyAmount int		  `json:"CurrencyAmount"`
	Status    string          `json:"status"`
	Name	  string		  `json:"Name"`
	Conn      *websocket.Conn `json:"-"` // Exclude Conn from JSON
}
