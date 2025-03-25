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
	"strconv"
	"time"
)

// OperatorModule defines the interface for operator-specific code
type OperatorInterface interface {
	HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, op models.Operator, rc *redisdb.RedisClient, pgs *postgrescli.PostgresCli)
	HandlePostBet(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, betValue int, gameID string) (int64, error)
	HandlePostWin(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, betValue int, gameID string) (int64, error)
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
	if err != nil {
		respondWithError(w, "Failed to fetch wallet", err)
		return
	}
	if logInResponse.Status != "success" {
		respondWithError(w, "Wallet request != success", fmt.Errorf("api err: %v", logInResponse.Data))
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
		respondWithError(w, "Failed to generate session", err)
		return
	}
	err = pgs.SaveSession(session)
	if err != nil {
		respondWithError(w, "Failed to save session", err)
		return
	}
	gameURL, err := generateGameURL(op.GameBaseUrl, req.Token, session.ID, logInResponse.Data.Currency)
	if err != nil {
		respondWithError(w, "Failed to generate game URL", err)
		return
	}
	// Final response
	response := models.SokkerDuelGamelaunchResponse{
		Token: req.Token,
		Url:   gameURL,
	}
	respondWithJSON(w, http.StatusOK, response)
}

func (m *SokkerDuelModule) HandlePostBet(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, betValue int, gameID string) (int64, error) {

	betData := models.SokkerDuelBet{
		OperatorGameName: session.OperatorIdentifier.GameName,
		Currency:         session.Currency,
		Amount:           betValue,
		TransactionID:    models.GenerateUUID(),
		RoundID:          gameID,
	}
	betResponse, err := walletrequests.SokkerDuelPostBet(session, betData)

	// Prepare the transaction record
	trans := models.Transaction{
		ID:        betData.TransactionID,
		SessionID: session.ID,
		Type:      "bet",
		Amount:    betValue,
		Currency:  session.Currency,
		Platform:  "sokkerpro",
		Operator:  "sokkerduel",
		Client:    session.PlayerName,
		Game:      session.OperatorIdentifier.GameName,
		RoundID:   gameID,
		Timestamp: time.Now(),
	}
	if err != nil {
		// If there's an error, set the status to "error" and store the error message
		trans.Status = "error"
		trans.Description = err.Error()
	} else {
		// If the API call is successful, set the status from the response
		trans.Status = betResponse.Status
		// Marshal the response data and store it in the transaction
		marshalResponseData, _ := json.Marshal(betResponse.Data)
		trans.Description = string(marshalResponseData)
	}
	if err := pgs.SaveTransaction(trans); err != nil {
		// If saving the transaction fails, return the error
		return -1, fmt.Errorf("failed to save transaction: %v", err)
	}
	session.ExtractID = betResponse.Data.ExtractID
	err = rc.AddSession(&session) // we save our session with the extract ID.
	// Return the API error if there was one
	if err != nil {
		return -1, fmt.Errorf("Failed to save session: %v", err)
	}

	fbalance, err := strconv.ParseFloat(betResponse.Data.Balance, 64)
	if err != nil {
		fmt.Println("Error:", err)
		return -1, err
	}
	fmt.Println("Float value:", fbalance)
	intBalance := int64(fbalance * 100.) // Convert to int after multiplying by 100

	// If everything is successful, return nil
	return intBalance, nil
}

func (m *SokkerDuelModule) HandlePostWin(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, betValue int, gameID string) (int64, error) {
	winData := models.SokkerDuelWin{
		OperatorGameName: session.OperatorIdentifier.GameName,
		Currency:         session.Currency,
		Amount:           betValue,
		TransactionID:    models.GenerateUUID(),
		RoundID:          gameID,
		ExtractID:        session.ExtractID,
	}
	betResponse, err := walletrequests.SokkerDuelPostWin(session, winData)

	// Prepare the transaction record
	trans := models.Transaction{
		ID:        winData.TransactionID,
		SessionID: session.ID,
		Type:      "win",
		Amount:    betValue,
		Currency:  session.Currency,
		Platform:  "sokkerpro",
		Operator:  "sokkerduel",
		Client:    session.PlayerName,
		Game:      session.OperatorIdentifier.GameName,
		RoundID:   gameID,
		Timestamp: time.Now(),
	}
	if err != nil {
		// If there's an error, set the status to "error" and store the error message
		trans.Status = "error"
		trans.Description = err.Error()
	} else {
		// If the API call is successful, set the status from the response
		trans.Status = betResponse.Status
		// Marshal the response data and store it in the transaction
		marshalResponseData, _ := json.Marshal(betResponse.Data)
		trans.Description = string(marshalResponseData)
	}
	if err := pgs.SaveTransaction(trans); err != nil {
		// If saving the transaction fails, return the error
		return -1, fmt.Errorf("failed to save transaction: %v", err)
	}
	session.ExtractID = 0
	rc.AddSession(&session) // we save our session with the extract ID.
	// Return the API error if there was one
	if err != nil {
		return -1, fmt.Errorf("API call failed: %v", err)
	}

	fbalance, err := strconv.ParseFloat(betResponse.Data.Balance, 64)
	if err != nil {
		fmt.Println("Error:", err)
		return -1, err
	}
	fmt.Println("Float value:", fbalance)
	intBalance := int64(fbalance * 100.) // Convert to int after multiplying by 100

	// If everything is successful, return nil
	return intBalance, nil
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

func generatePlayerSession(op models.Operator, token, username, currency string, balance float64, rc *redisdb.RedisClient) (models.Session, error) {
	session := models.Session{
		ID:         models.GenerateUUID(),
		Token:      token,
		PlayerName: username,
		Balance:    int64(balance),
		Currency:   currency,
		OperatorIdentifier: models.OperatorIdentifier{
			OperatorName:     op.OperatorName,
			OperatorGameName: op.OperatorGameName,
			GameName:         op.GameName,
		},
		OperatorBaseUrl: op.OperatorWalletBaseUrl,
		CreatedAt:       time.Now(),
	}
	err := rc.AddSession(&session)
	return session, err
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
