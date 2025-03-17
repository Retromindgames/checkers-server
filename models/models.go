package models

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID              string    `json:"session_id"`
	Token           string    `json:"token"`
	PlayerName      string    `json:"player_name"`
	Balance         int64     `json:"balance"`
	Currency        string    `json:"currency"`
	OperatorBaseUrl string    `json:"operator_base_url"`
	CreatedAt       time.Time `json:"created_at"`
	OperatorIdentifier OperatorIdentifier `json:"operator_identifier"`
}

type OperatorIdentifier struct {
	OperatorName	 string  `json:"operator_name"`
	OperatorGameName string	 `json:"operator_game_name"`
	GameName		 string	 `json:"game_name"`
}

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
