package models

import (
	"fmt"

	"github.com/gorilla/websocket"
)

type PlayerStatus string

const (
	StatusOffline               PlayerStatus = "OFFLINE"
	StatusOnline                PlayerStatus = "ONLINE"
	StatusInQueue               PlayerStatus = "IN_QUEUE"
	StatusAwaitingReady         PlayerStatus = "AWAITING_READY"
	StatusAwaitingOponenteReady PlayerStatus = "AWAITING_OPONENTE_READY"
	StatusInGame                PlayerStatus = "IN_GAME"
)

type Player struct {
	ID             string          `json:"id"`
	Token          string          `json:"token"`
	RoomID         string          `json:"room_id"`
	SessionID      string          `json:"session_id"`
	Currency       string          `json:"currency"`
	CurrencyAmount int             `json:"currency_amount"`
	Status         PlayerStatus    `json:"status"`
	SelectedBet    float64         `json:"selected_bet"`
	Name           string          `json:"name"`
	Conn           *websocket.Conn `json:"-"` // Exclude Conn from JSON
}

// This map will hold the valid status transition
var validStatusTransitions = map[PlayerStatus]map[PlayerStatus]bool{
	StatusOffline: {
		StatusOnline: true,
	},
	StatusOnline: {
		StatusOffline: true,
		StatusInQueue: true,
	},
	StatusInQueue: {
		StatusOnline:        true,
		StatusAwaitingReady: true,
	},
	StatusAwaitingReady: {
		StatusInQueue:               true,
		StatusAwaitingOponenteReady: true,
	},
	StatusAwaitingOponenteReady: {
		StatusAwaitingReady: true,
		StatusInGame:        true,
	},
	StatusInGame: {
		StatusOnline: true,
	},
}

// This updates and checks that our player status is the right one.
func (p *Player) UpdatePlayerStatus(newStatus PlayerStatus) error {
	if p.Status == newStatus {
		return fmt.Errorf("player is already in status %s", newStatus)
	}

	if valid, ok := validStatusTransitions[PlayerStatus(p.Status)][newStatus]; !ok || !valid {
		return fmt.Errorf("invalid status transition from %s to %s", p.Status, newStatus)
	}

	p.Status = newStatus
	return nil
}
