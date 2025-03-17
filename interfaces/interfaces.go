package interfaces

import (
	"checkers-server/models"
	"checkers-server/postgrescli"
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
	HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, op models.Operator, rc *redisdb.RedisClient, pgs *postgrescli.PostgresCli)
	HandlePostToWallet(pgs *postgrescli.PostgresCli, session models.Session, betValue int, gameID string) error
}

// OperatorModules maps operator names to their respective modules
var OperatorModules = map[string]OperatorInterface{
	"SokkerDuel": &SokkerDuelModule{},
	//"AnotherOperator": &AnotherOperatorModule{},
	// Add more operators as needed
}

// SokkerDuelModule handles requests for the SokkerDuel operator
type SokkerDuelModule struct{}

func (m *SokkerDuelModule) HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, op models.Operator, rc *redisdb.RedisClient, pgs *postgrescli.PostgresCli) {
	// Fetch wallet information
	logInResponse, err := walletrequests.SokkerDuelGetWallet(op, req.Token)
	if err != nil || logInResponse.Status != status {
		http.Error(w, fmt.Sprintf("Failed to fetch wallet: %v, api err:%v", err, logInResponse.Data), http.StatusInternalServerError)
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
	err = pgs.SaveSession(session)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save session to postgres session: %v", err), http.StatusInternalServerError)
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

func (m *SokkerDuelModule) HandlePostToWallet(pgs *postgrescli.PostgresCli, session models.Session, betValue int, gameID string) (error) {
	
	betData := models.SokkerDuelBet {
		OperatorGameName: session.OperatorIdentifier.GameName,
		Currency: session.Currency,
		Amount: int(betValue),
		TransactionID: models.GenerateUUID(),
	}
	
	betResponse, err := walletrequests.SokkerDuelPostBet(session, betData)
	trans := models.Transaction {
		ID: betData.TransactionID,
		SessionID: session.ID,
		Type: "bet",
		Amount: betValue,
		Currency: session.Currency,
		Platform: "sokkerpro",
		Operator: "sokkerduel",
		Client: session.PlayerName,
		Game: session.OperatorIdentifier.GameName,
		Status: betResponse.Status,
		Description: betResponse.Data,
		RoundID: gameID,
		Timestamp: time.Now(),
	}
	if err != nil || betResponse.Status != "success"{
		
	}	
	
	pgs.SaveTransaction(trans)
	return fmt.Errorf("")
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
		OperatorIdentifier: models.OperatorIdentifier{
			OperatorName:    	 op.OperatorName,
			OperatorGameName:    op.OperatorGameName,
			GameName:    		 op.GameName,

		},
		OperatorBaseUrl: op.OperatorWalletBaseUrl,
		CreatedAt:       time.Now(),
	}
	err := rc.AddSession(&session)
	return session, err
}
