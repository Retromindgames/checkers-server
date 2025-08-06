package models

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Lavizord/checkers-server/postgrescli/ent"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type GameLaunchResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

/*
type Session struct {
	ID              string      `json:"session_id"`
	Token           string      `json:"token"`
	PlayerName      string      `json:"player_name"`
	Currency        string      `json:"currency"`
	OperatorBaseUrl string      `json:"operator_base_url"`
	CreatedAt       time.Time   `json:"created_at"`
	ExtractID       int64       `json:"extract_id"` // This was created to store the extract ID of a bet, so that we can later use it in the win post...
	OperatorDTO     OperatorDTO `json:"operator"`
}
*/

type Session struct {
	ID                string    `json:"session_id"`
	Token             string    `json:"token"`
	ClientID          string    `json:"client_id"`
	Demo              bool      `json:"demo"`
	OperatorID        int       `json:"operator_id"`
	GameID            int       `json:"game_id"`
	GameVersionID     int       `json:"game_version_id"`
	MathVersionID     int       `json:"math_version_id"`
	CurrencyVersionID int       `json:"currency_version_id"`
	CreatedAt         time.Time `json:"created_at"`
	DeletedAt         time.Time `json:"deleted_at"`
	ExtractID         int64     `json:"extract_id"` // This was created to store the extract ID of a bet, so that we can later use it in the win post...
}

func (s *Session) IsTokenExpired() bool {
	token, _, err := new(jwt.Parser).ParseUnverified(s.Token, jwt.MapClaims{})
	if err != nil {
		log.Println("Error parsing token:", err)
		return true // Assume expired if we can't parse
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Println("Invalid token claims")
		return true
	}

	// Extract expiration time (exp)
	expFloat, ok := claims["exp"].(float64)
	if !ok {
		log.Println("Expiration claim missing")
		return true
	}

	expTime := time.Unix(int64(expFloat), 0)
	return time.Now().After(expTime)
}

type Transaction struct {
	ID          string    `json:"transaction_id"` // Unique ID for each transaction
	SessionID   string    `json:"session_id"`     // Session ID for the player
	Type        string    `json:"type"`           // Type of transaction: 'bet' or 'win'
	Amount      int64     `json:"amount"`         // Amount in cents
	Currency    string    `json:"currency"`       // Currency code (e.g., EUR, USD)
	Platform    string    `json:"platform"`       // Platform name
	Operator    string    `json:"operator"`       // Operator name (e.g., SokkerDuel)
	Client      string    `json:"client"`         // Client ID (player ID)
	Game        string    `json:"game"`           // Internal game name
	Status      string    `json:"status"`         // HTTP status code
	Description string    `json:"description"`    // Description (e.g., "Insufficient Funds" or "OK")
	RoundID     string    `json:"round_id"`       // Foreign key to the round / game
	Timestamp   time.Time `json:"timestamp"`      // Timestamp in UTC
}

type OperatorDTO struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Alias string `json:"operator_game_name"`
}

func OperatorToDTO(op *ent.Operator) OperatorDTO {
	return OperatorDTO{
		Name:  op.Name,
		Alias: op.Alias,
	}
}
func OperatorsToDTO(ops []*ent.Operator) []OperatorDTO {
	dto := make([]OperatorDTO, len(ops))
	for i, op := range ops {
		dto[i] = OperatorToDTO(op)
	}
	return dto
}

// Flattens the data from the game config and related entities into a single object.
//
// This will be stored in redis too avoid too many querys to the database.
type GameConfigDTO struct {
	// Operator
	OperatorID    int
	OperatorName  string
	OperatorAlias string
	// Platform
	PlatformName              string
	PlatformHash              string
	PlatformHomeButtonPayload string
	// Game
	GameID            int
	GameName          string
	GameTrademarkName string
	// Game Versions
	GameVersionID  int
	GameVersion    string
	UrlMediaPack   string
	UrlReleaseNote string
	UrlGameManual  string
	// Game Configs
	CanDemo        bool
	CanTournament  bool
	CanFreeBets    bool
	CanDropAndWins bool
	CanBuyBonus    bool
	CanTurbo       bool
	CanAutoBet     bool
	CanAutoCashout bool
	CanAnteBet     bool
	// Currency Versions
	CurrencyVersionID          int
	CurrencyName               string
	CurrencySymbol             string
	CurrencyThousandSeparator  string
	CurrencyUnitsSeparator     string
	CurrencySymbolPosition     string
	CurrencyDenominator        int
	CurrencyMinBet             int
	CurrencyMaxExp             int
	CurrencyDefaultMultiplier  int
	CurrencyCrashBetIncrement  int
	CurrencySlotsBetMultiplier []int
	// Math Versions
	MathVersionID             int
	MathName                  string
	MathVersion               string
	MathVersionUrlReleaseNote string
	Volatility                int
	Rtp                       int
	MaxWin                    int
	//CanBuyBonus      int //TODO: This field is in several tables....
	//CanAnteBet       int //TODO: This field is in several tables....
}

func GameConfigToDTO(gc *ent.GameConfig) GameConfigDTO {
	dto := GameConfigDTO{}
	b, _ := json.MarshalIndent(gc, "", "  ")
	fmt.Println(string(b))

	if gc.Edges.Operator != nil {
		dto.OperatorID = gc.Edges.Operator.ID
		dto.OperatorName = gc.Edges.Operator.Name
		dto.OperatorAlias = gc.Edges.Operator.Alias

		if gc.Edges.Operator.Edges.Platforms != nil {
			dto.PlatformName = gc.Edges.Operator.Edges.Platforms.Name
			dto.PlatformHash = gc.Edges.Operator.Edges.Platforms.Hash
			dto.PlatformHomeButtonPayload = gc.Edges.Operator.Edges.Platforms.HomeButtonPayload
		}
	}

	if gc.Edges.GameVersions != nil {
		dto.GameVersionID = gc.Edges.GameVersions.ID
		dto.GameID = gc.Edges.Games.ID
		dto.GameName = gc.Edges.Games.Name
		dto.GameTrademarkName = gc.Edges.Games.TrademarkName
		dto.GameVersion = gc.Edges.GameVersions.Version
		dto.UrlMediaPack = gc.Edges.GameVersions.URLMediaPack
		dto.UrlReleaseNote = gc.Edges.GameVersions.URLReleaseNote
		dto.UrlGameManual = gc.Edges.GameVersions.URLMediaPack
	}

	if gc.Edges.CurrencyVersions != nil {
		dto.CurrencyVersionID = gc.Edges.CurrencyVersions.ID
		dto.CurrencyName = gc.Edges.CurrencyVersions.Name
		dto.CurrencyDenominator = gc.Edges.CurrencyVersions.Denominator
		dto.CurrencyMinBet = gc.Edges.CurrencyVersions.MinBet
		dto.CurrencyMaxExp = gc.Edges.CurrencyVersions.MaxExp
		dto.CurrencyDefaultMultiplier = gc.Edges.CurrencyVersions.DefaultMultiplier
		dto.CurrencyCrashBetIncrement = gc.Edges.CurrencyVersions.CrashBetIncrement
		dto.CurrencySlotsBetMultiplier = gc.Edges.CurrencyVersions.SlotsBetMultipliers

		if gc.Edges.CurrencyVersions.Edges.Currency != nil {
			dto.CurrencySymbol = gc.Edges.CurrencyVersions.Edges.Currency.Symbol
			dto.CurrencySymbolPosition = gc.Edges.CurrencyVersions.Edges.Currency.SymbolPosition
			dto.CurrencyThousandSeparator = gc.Edges.CurrencyVersions.Edges.Currency.ThousandsSeparator
			dto.CurrencyUnitsSeparator = gc.Edges.CurrencyVersions.Edges.Currency.UnitsSeparator
		}
	}

	if gc.Edges.MathVersions != nil {
		dto.MathVersionID = gc.Edges.MathVersions.ID
		dto.MathName = gc.Edges.MathVersions.Name
		dto.MathVersion = gc.Edges.MathVersions.Version
		dto.MathVersionUrlReleaseNote = gc.Edges.MathVersions.URLReleaseNote
		dto.Volatility = gc.Edges.MathVersions.Volatility
		dto.Rtp = gc.Edges.MathVersions.Rtp
		dto.MaxWin = gc.Edges.MathVersions.MaxWin
	}

	dto.CanDemo = gc.CanDemo
	dto.CanTournament = gc.CanTournament
	dto.CanFreeBets = gc.CanFreeBets
	dto.CanDropAndWins = gc.CanDropAndWins
	dto.CanBuyBonus = gc.CanBuyBonus
	dto.CanTurbo = gc.CanTurbo
	dto.CanAutoBet = gc.CanAutoBet
	dto.CanAutoCashout = gc.CanAutoCashout
	dto.CanAnteBet = gc.CanAnteBet

	return dto
}

func GameConfigsToDTO(gcs []*ent.GameConfig) []GameConfigDTO {
	dto := make([]GameConfigDTO, len(gcs))
	for i, gc := range gcs {
		dto[i] = GameConfigToDTO(gc)
	}
	return dto
}

type PlayerCountPerBetValue struct {
	BetValue    float64 `json:"bet_value"`
	PlayerCount int64   `json:"player_count"`
}

type QueueNumbersResponse struct {
	QueuNumbers []PlayerCountPerBetValue `json:"player_count_per_bet_value"`
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
	Color         int     `json:"color"`
	Opponent      string  `json:"opponent"`
	RoomID        string  `json:"room_id"`
	Winnings      float64 `json:"winnings"`
	Timer         int     `json:"timer"`
	PlayerReady   bool    `json:"player_ready"`
	OpponentReady bool    `json:"opponent_ready"`
}

type QueueConfirmation struct {
	IsConfirmed bool `json:"is_confirmed"`
}

func GenerateUUID() string {
	return uuid.New().String() // Generates a new UUID and returns it as a string
}
