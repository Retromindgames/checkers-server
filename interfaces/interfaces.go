package interfaces

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

// PlatformInterface defines the interface for platform-specific code
type PlatformInterface interface {
	HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, gc *models.GameConfigDTO, rc *redisdb.RedisClient, pgs *postgrescli.PostgresCli)
	HandleFetchWalletBalance(s models.Session, rc *redisdb.RedisClient) (int64, error)
	HandlePostBet(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, betValue int64, gameID string) (int64, error)
	HandlePostWin(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, betValue int64, gameID string) (int64, int64, error)
}

// PlatformModules maps platforms names to their respective modules
var PlatformModules = map[string]PlatformInterface{
	"SokkerDuel": &SokkerDuelModule{},
	"TestOp":     &TestModule{},
}

// SokkerDuelModule handles requests for the SokkerDuel platform
type SokkerDuelModule struct{}

// TestModule handles requests for test accounts
type TestModule struct{}

// Helper function to save failed transactions
func saveFailedBetTransaction(pgs *postgrescli.PostgresCli, session models.Session, betData models.SokkerDuelBet, apiError error, gameID string) error {
	ctx := context.Background()
	_, err := pgs.EntCli.Transaction.
		Create().
		SetType("bet").
		SetToken(session.Token).
		SetCurrency(session.)
		Save(ctx)

	if err != nil {
		log.Printf("[PostgresCli] - error saving session: %v", err)
		return fmt.Errorf("ent: save session: %w", err)
	}
	return nil
}

func saveFailedWinTransaction(pgs *postgrescli.PostgresCli, session models.Session, winData models.SokkerDuelWin, apiError error, gameID string) error {
	trans := models.Transaction{
		ID:        winData.TransactionID,
		SessionID: session.ID,
		Type:      "win",
		Amount:    winData.Amount,
		Currency:  session.Currency,
		Platform:  "sokkerpro",
		Operator:  "SokkerDuel",
		Client:    session.PlayerName,
		//Game:        session.OperatorIdentifier.GameName,
		RoundID:     gameID,
		Timestamp:   time.Now(),
		Status:      "error",
		Description: apiError.Error(),
	}
	return pgs.SaveTransaction(trans)
}

func generateGameURL(baseURL, token, sessionID, currency string) (string, error) {
	// Parse the base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %v", err)
	}

	// Add query parameters
	query := url.Values{}
	query.Add("token", token)
	query.Add("sessionid", sessionID)
	query.Add("currency", currency)
	parsedURL.RawQuery = query.Encode()

	// Return the full URL as a string
	return parsedURL.String(), nil
}

func generatePlayerSession(req models.GameLaunchRequest, gc *models.GameConfigDTO, username string, rc *redisdb.RedisClient) (*models.Session, error) {
	session := models.Session{
		ID:                models.GenerateUUID(),
		Token:             req.Token,
		ClientID:          username,
		Demo:              req.Demo,
		OperatorID:        gc.OperatorID,
		GameID:            gc.GameID,
		GameVersionID:     gc.GameVersionID,
		MathVersionID:     gc.MathVersionID,
		CurrencyVersionID: gc.CurrencyVersionID,
		CreatedAt:         time.Now(),
	}
	err := rc.AddSession(&session)
	return &session, err
}

func checkExistingSession(token string, rc *redisdb.RedisClient) (*models.Session, error) {
	// First, check Redis for an active session
	session, err := rc.GetSessionByToken(token)
	if err == nil && session != nil {
		return session, nil // Session exists
	}
	return nil, fmt.Errorf("session not found")
}

func checkPreviousPlayerSession(operator string, playerName string, currency string, rc *redisdb.RedisClient) (*models.Session, error) {
	//fmt.Printf("Checking previous Player session: %v, %v, %v.", operator, playerName, currency)
	session, err := rc.GetSessionByOperatorPlayerCurrency(operator, playerName, currency)
	if err == nil && session != nil {
		log.Printf("Session found!: ID:%v", session.ID)
		return session, nil // Session exists
	}
	//fmt.Printf("Session not found.")
	return nil, fmt.Errorf("session not found")
}

// Helper function to send JSON errors
func respondWithError(w http.ResponseWriter, message string, err error) {
	respondWithJSON(w, http.StatusInternalServerError, map[string]string{
		"error":   message,
		"details": err.Error(),
	})
}

// Helper function to send JSON responses
func respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(payload)
}

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("failed to marshal API response: %v", err)
		return []byte("")
	}
	return b
}

func CalculateWinAmount(betValue int64, winFactor float64) int64 {
	// Multiply by 2 then by 0.9 (equivalent to multiplying by 1.8)
	// Using float64 for precise multiplication then converting back to int
	winAmount := float64(betValue*2) * winFactor
	//log.Printf("Calculating win amount:[betValue: %v], [winFactor: %v], final value: [%v]", betValue, winFactor, winAmount)
	return int64(winAmount) // Truncates decimal places
}

func handleSaveSession(session *models.Session, pgs *postgrescli.PostgresCli) {
	err := pgs.SaveSession(*session)
	if err != nil {
		return
	}
}
