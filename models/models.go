package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Player struct {
	ID        string          `json:"id"`
	Token     string          `json:"token"`
	SessionID string          `json:"sessionid"`
	Currency  string          `json:"currency"`
	CurrencyAmount int		  `json:"CurrencyAmount"`
	Status    string          `json:"status"`	// IN GAME! ONLINE! OFFLNE
	SelectedBid float64		  `json:"SelectedBid"`
	Name	  string		  `json:"Name"`
	Conn      *websocket.Conn `json:"-"` // Exclude Conn from JSON
}

// Room represents the data for a game room, containing two players and game details
type Room struct {
	ID        		  string     `json:"ID"`
	Player1           *Player    `json:"Player1"`
	Player2           *Player    `json:"Player2"`
	StartDate         time.Time  `json:"StartDate"`
	Currency          string     `json:"Currency"`    		// Currency for the room
	BidAmount         float64    `json:"BidAmount"`  		// Bet amount for the game
	Turn	          int    	 `json:"Turn"`
	CurrentTurnPlayer *Player    `json:"CurrentTurnPlayer"`
	GameBoard         string     `json:"GameBoard"`  		// TODO: Representation of the game board (could be a string of board state?)
	GameStatus        string     `json:"GameStatus"` 		// Status of the game (e.g., "waiting", "in_progress", "finished")
	GameEndDate       *time.Time `json:"GameEndDate,omitempty"`
	Winner            *Player    `json:"Winner,omitempty"`  // Player who won (if game is over)
	IsRoomOpen     	  bool       `json:"IsRoomOpen"`
}

type RoomAgregate struct {
	Currency          string     `json:"Currency"`    		// Currency for the room
	BidAmount         float64    `json:"BidAmount"`  		// Bet amount for the game
	GameStatus        string     `json:"GameStatus"` 		// Status of the game (e.g., "waiting", "in_progress", "finished")
	GameEndDate       *time.Time `json:"GameEndDate,omitempty"`
}


type RoomValue struct {
	ID        string `json:"id"`
	Player     string `json:"name"`
	Currency  string `json:"currency"`
	SelectedBid float64 `json:"SelectedBid"`
}

type CreateRoomMessage struct {
	Command string `json:"command"`
	Value RoomValue `json:"value"`
}

func GenerateUUID() string {
	return uuid.New().String() // Generates a new UUID and returns it as a string
}
