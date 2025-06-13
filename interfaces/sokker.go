package interfaces

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
	"github.com/Lavizord/checkers-server/walletrequests"
)

func (m *SokkerDuelModule) HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, op models.Operator, rc *redisdb.RedisClient, pgs *postgrescli.PostgresCli) {
	// Fetch wallet information
	logInResponse, err := walletrequests.SokkerDuelGetWallet(op.OperatorWalletBaseUrl, req.Token)
	if err != nil {
		respondWithError(w, "Failed to fetch wallet", err)
		return
	}
	if logInResponse.Status != "success" {
		respondWithError(w, "Wallet request != success", fmt.Errorf("api err: %v", logInResponse.Data))
		return
	}
	session, err := checkExistingSession(req.Token, rc)
	if err != nil || session == nil {
		session, _ = checkPreviousPlayerSession(req.OperatorName, logInResponse.Data.Username, req.Currency, rc)
		if session != nil {
			rc.DisconnectPlayer(session.ID) // We send a message to disconnect the previous websocket connection.
			rc.RemoveSession(session.ID)    // If the session exists, from a previous token, we remove the session
			session.Token = req.Token       // We just update the token.
			rc.AddSession(session)          // we update the session in redis.
		} else {
			session, err = generatePlayerSession( // then we generate a new session.
				op,
				req.Token,
				logInResponse.Data.Username,
				logInResponse.Data.Currency,
				rc,
			)
			if err != nil {
				respondWithError(w, "Failed to generate session", err)
				return
			}
			go handleSaveSession(session, pgs)
		}
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

func (m *SokkerDuelModule) HandleFetchWalletBalance(s models.Session, rc *redisdb.RedisClient) (int64, error) {
	logInResponse, err := walletrequests.SokkerDuelGetWallet(s.OperatorBaseUrl, s.Token)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch wallet: %v", err)
	}
	return int64(logInResponse.Data.Balance), nil
}

func (m *SokkerDuelModule) HandlePostBet(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, betValue int64, gameID string) (int64, error) {
	// Validate input parameters
	if betValue <= 0 {
		return -1, fmt.Errorf("invalid bet value: %d", betValue)
	}
	if gameID == "" {
		return -1, fmt.Errorf("empty game ID")
	}
	if session.ID == "" {
		return -1, fmt.Errorf("invalid session")
	}

	betData := models.SokkerDuelBet{
		OperatorGameName: session.OperatorIdentifier.GameName,
		Currency:         session.Currency,
		Amount:           betValue,
		TransactionID:    models.GenerateUUID(),
		RoundID:          gameID,
	}
	// Make API call - now we know it either returns success response or error
	betResponse, err := walletrequests.SokkerDuelPostBet(session, betData)
	if err != nil {
		if saveErr := saveFailedBetTransaction(pgs, session, betData, err, gameID); saveErr != nil {
			return -1, fmt.Errorf("API error: %v | Transaction save error: %v", err, saveErr)
		}
		return -1, err // Return original API error
	}

	// At this point, we're guaranteed betResponse is valid and status="success"
	// Prepare and save transaction
	trans := models.Transaction{
		ID:          betData.TransactionID,
		SessionID:   session.ID,
		Type:        "bet",
		Amount:      betValue,
		Currency:    session.Currency,
		Platform:    "sokkerpro",
		Operator:    "SokkerDuel",
		Client:      session.PlayerName,
		Game:        session.OperatorIdentifier.GameName,
		RoundID:     gameID,
		Timestamp:   time.Now(),
		Status:      betResponse.Status,
		Description: string(mustMarshal(betResponse)), // Safe because we know Data exists
	}
	go pgs.SaveTransaction(trans)

	// Update session
	session.ExtractID = betResponse.Data.ExtractID
	if err := rc.AddSession(&session); err != nil {
		return -1, fmt.Errorf("failed to save session: %v", err)
	}
	// Parse balance (we know it exists from API contract)
	fbalance, err := strconv.ParseFloat(betResponse.Data.Balance, 64)
	if err != nil {
		return -1, fmt.Errorf("failed to parse balance: %v", err)
	}
	return int64(fbalance * 100), nil
}

func (m *SokkerDuelModule) HandlePostWin(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, winValue int64, gameID string) (int64, int64, error) {
	// Validate input parameters
	if winValue <= 0 {
		return -1, -1, fmt.Errorf("invalid win value: %d", winValue)
	}
	if gameID == "" {
		return -1, -1, fmt.Errorf("empty game ID")
	}
	if session.ID == "" {
		return -1, -1, fmt.Errorf("invalid session")
	}
	var winnings = CalculateWinAmount(winValue, session.OperatorIdentifier.WinFactor)
	winData := models.SokkerDuelWin{
		OperatorGameName: session.OperatorIdentifier.GameName,
		Currency:         session.Currency,
		Amount:           winnings,
		TransactionID:    models.GenerateUUID(),
		RoundID:          gameID,
		ExtractID:        session.ExtractID,
	}
	// Make API call - guaranteed to return either success response or error
	winResponse, err := walletrequests.SokkerDuelPostWin(session, winData)
	if err != nil {
		// Save failed transaction before returning
		go saveFailedWinTransaction(pgs, session, winData, err, gameID)
		return -1, winnings, err // Return original API error
	}
	// At this point, we're guaranteed winResponse is valid and status="success"
	trans := models.Transaction{
		ID:          winData.TransactionID,
		SessionID:   session.ID,
		Type:        "win",
		Amount:      winnings,
		Currency:    session.Currency,
		Platform:    "sokkerpro",
		Operator:    "SokkerDuel",
		Client:      session.PlayerName,
		Game:        session.OperatorIdentifier.GameName,
		RoundID:     gameID,
		Timestamp:   time.Now(),
		Status:      winResponse.Status,
		Description: string(mustMarshal(winResponse)), // Safe because we know Data exists
	}
	go pgs.SaveTransaction(trans)
	// Reset ExtractID in session
	session.ExtractID = 0
	if err := rc.AddSession(&session); err != nil {
		return -1, -1, fmt.Errorf("failed to save session: %v", err)
	}
	// Parse balance (we know it exists from API contract)
	fbalance, err := strconv.ParseFloat(winResponse.Data.Balance, 64)
	if err != nil {
		return -1, -1, fmt.Errorf("failed to parse balance: %v", err)
	}
	return int64(fbalance * 100), winnings, nil
}
