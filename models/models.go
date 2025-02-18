package models

import (
	"github.com/google/uuid"
)


type RoomAggregate struct {
	AggregateValue float64 `json:"aggregate_value"`
	Count          int64   `json:"count"`
}

type RoomAggregateResponse struct {
	PlayersWaiting int             `json:"players_waiting"`
	RoomAggregate  []RoomAggregate `json:"room_aggregate"`
}

type RoomValue struct {
	ID       string  `json:"id"`
	Player   string  `json:"name"`
	Currency string  `json:"currency"`
	BetValue float64 `json:"bet_value"`
}

type CreateRoomMessage struct {
	Command string    `json:"command"`
	Value   RoomValue `json:"value"`
}

type PairedValue struct {
	Color    int    `json:"color"`
	Opponent string `json:"opponent"`
	RoomID   string `json:"room_id"`
}

type QueueConfirmation struct {
	IsConfirmed bool `json:"is_confirmed"`
}

func GenerateUUID() string {
	return uuid.New().String() // Generates a new UUID and returns it as a string
}
