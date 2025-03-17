package interfaces

import (
	"checkers-server/models"
	"checkers-server/redisdb"
	"checkers-server/walletrequests"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// OperatorModule defines the interface for operator-specific code
type OperatorInterface interface {
	HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, op models.Operator, rc *redisdb.RedisClient)
}

// OperatorModules maps operator names to their respective modules
var OperatorModules = map[string]OperatorInterface{
	"SokkerDuel": &SokkerDuelModule{},
	//"AnotherOperator": &AnotherOperatorModule{},
	// Add more operators as needed
}

// SokkerDuelModule handles requests for the SokkerDuel operator
type SokkerDuelModule struct{}

func (m *SokkerDuelModule) HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, op models.Operator, rc *redisdb.RedisClient) {
	// Fetch wallet information
	logInResponse, err := walletrequests.SokkerDuelGetWallet(op, req.Token)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch wallet: %v", err), http.StatusInternalServerError)
		return
	}

	session, err := generatePlayerSession(
		op,
		req.Token,
		logInResponse.Data.Username,
		logInResponse.Data.Currency,
		logInResponse.Data.Balance,
		rc,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate session: %v", err), http.StatusInternalServerError)
		return
	}

	gameURL, err := generateGameURL(op.GameBaseUrl, req.Token, session.ID, logInResponse.Data.Currency)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate game URL: %v", err), http.StatusInternalServerError)
		return
	}

	response := models.SokkerDuelGamelaunchResponse{
		Token: req.Token,
		Url:   gameURL,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Helper function to generate the game URL
func generateGameURL(baseURL, token, sessionID, currency string) (string, error) {
	// Parse the base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %v", err)
	}

	// Add query parameters
	query := url.Values{}
	query.Add("token", token)
	query.Add("sessionId", sessionID)
	query.Add("currency", currency)
	parsedURL.RawQuery = query.Encode()

	// Return the full URL as a string
	return parsedURL.String(), nil
}

func generatePlayerSession(op models.Operator, token, username, currency string, balance int64, rc *redisdb.RedisClient) (models.Session, error) {
	session := models.Session{
		ID:              models.GenerateUUID(),
		Token:           token,
		PlayerName:      username,
		Balance:         balance,
		Currency:        currency,
		OperatorName:    op.OperatorName,
		OperatorBaseUrl: op.OperatorWalletBaseUrl,
		CreatedAt:       time.Now(),
	}
	err := rc.AddSession(&session)
	return session, err
}
