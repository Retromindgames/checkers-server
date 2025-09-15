package models

import (
	"time"
)

type Transaction struct {
	ID           string    `json:"transaction_id"` // Unique ID for each transaction
	Type         string    `json:"type"`           // Type of transaction: 'bet' or 'win'
	Amount       int64     `json:"amount"`         // Amount in cents
	Currency     string    `json:"currency"`       // Currency code (e.g., EUR, USD)
	Platform     string    `json:"platform"`       // Platform name
	Operator     string    `json:"operator"`       // Operator name (e.g., SokkerDuel)
	Client       string    `json:"client"`         // Client ID (player ID)
	Game         string    `json:"game"`           // Internal game name
	Status       int       `json:"status"`         // HTTP status code
	Description  string    `json:"description"`    // Description (e.g., "Insufficient Funds" or "OK")
	Timestamp    time.Time `json:"timestamp"`      // Timestamp in UTC
	MathProfile  string    `json:"math_profile"`   // ?
	Denominator  int       `json:"denominator"`    //
	FinalBalance int       `json:"final_balance"`  //
	SeqID        int       `json:"seq_id"`         //
	Multiplier   int       `json:"multiplier"`     // ?
	SessionID    string    `json:"session_id"`     // Session ID for the player
	RoundID      string    `json:"round_id"`       // Foreign key to the round / game
}

type BetData struct {
	OperatorGameName string `json:"game_id"`
	Currency         string `json:"currency"`
	Amount           int64  `json:"amount"`
	TransactionID    string `json:"transaction_id"`
	RoundID          string `json:"round_id"`
}

// TODO: Finish this. This is to be used before saving a new transaction, we create a blank
// one and fill in the rest of the data before saving it to postgres.
func NewTransaction(s Session, b BetData) Transaction {
	trans := Transaction{
		ID:        GenerateUUID(),
		Type:      "bet",
		Amount:    b.Amount,
		Currency:  s.Currency,
		Platform:  "sokkerpro",
		Operator:  "SokkerDuel",
		Client:    s.ClientID,
		Game:      b.OperatorGameName,
		Timestamp: time.Now(),
		RoundID:   b.RoundID,
		SessionID: s.ID,
	}
	return trans
}
