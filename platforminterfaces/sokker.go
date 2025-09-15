package platforminterfaces

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/platforminterfaces/walletrequests"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

func (m *SokkerDuelModule) HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, gc *models.GameConfigDTO, rc *redisdb.RedisClient, pgs *postgrescli.PostgresCli) {
	baseUrl := os.Getenv("SOKKER_GAME_URL")

	// Fetch wallet information
	logInResponse, err := walletrequests.SokkerDuelGetWallet(baseUrl, req.Token)
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
				req, gc, req.Token, rc,
			)
			if err != nil {
				respondWithError(w, "Failed to generate session", err)
				return
			}
			go handleSaveSession(session, pgs)
		}
	}
	gameURL, err := generateGameURL(baseUrl, req.Token, session.ID, logInResponse.Data.Currency)
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

func (m *SokkerDuelModule) HandleFetchWalletBalance(s models.Session, gc *models.GameConfigDTO, rc *redisdb.RedisClient) (int, error) {
	baseUrl := os.Getenv("SOKKER_GAME_URL")

	logInResponse, err := walletrequests.SokkerDuelGetWallet(baseUrl, s.Token)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch wallet: %v", err)
	}
	return int(logInResponse.Data.Balance), nil
}

func (m *SokkerDuelModule) HandlePostBet(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, gc *models.GameConfigDTO, betAmount int, roundID string) (int, error) {
	baseUrl := os.Getenv("SOKKER_GAME_URL")

	// Validate input parameters
	if betAmount <= 0 {
		return -1, fmt.Errorf("invalid bet value: %d", betAmount)
	}
	if roundID == "" {
		return -1, fmt.Errorf("empty game ID")
	}
	if session.ID == "" {
		return -1, fmt.Errorf("invalid session")
	}

	betData := models.SokkerDuelBet{
		//OperatorGameName: session.OperatorIdentifier.GameName,
		Currency:      session.Currency,
		Amount:        betAmount,
		TransactionID: models.GenerateUUID(),
		RoundID:       roundID,
	}
	// Make API call - now we know it either returns success response or error
	betResponse, err := walletrequests.SokkerDuelPostBet(baseUrl, session.Token, betData)
	if err != nil {
		go pgs.SaveTransaction(
			session,
			*gc,
			nil,
			"bet",
			0,
			betAmount,
			99999,
			9999-betAmount,
		)
		return -1, err // Return original API error
	}

	go pgs.SaveTransaction(
		session,
		*gc,
		nil,
		"bet",
		200,
		betAmount,
		99999,
		9999-betAmount,
	)

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
	return int(fbalance * 100), nil
}

func (m *SokkerDuelModule) HandlePostWin(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, gc *models.GameConfigDTO, winValue int, roundID string) (int, int, error) {
	baseUrl := os.Getenv("SOKKER_GAME_URL")

	// Validate input parameters
	if winValue <= 0 {
		return -1, -1, fmt.Errorf("invalid win value: %d", winValue)
	}
	if roundID == "" {
		return -1, -1, fmt.Errorf("empty game ID")
	}
	if session.ID == "" {
		return -1, -1, fmt.Errorf("invalid session")
	}
	//var winnings = CalculateWinAmount(winValue, session.OperatorIdentifier.WinFactor)
	winData := models.SokkerDuelWin{
		//OperatorGameName: session.OperatorIdentifier.GameName,
		Currency: session.Currency,
		//Amount:           winnings,
		TransactionID: models.GenerateUUID(),
		RoundID:       roundID,
		ExtractID:     session.ExtractID,
	}
	// Make API call - guaranteed to return either success response or error
	winResponse, err := walletrequests.SokkerDuelPostWin(baseUrl, session.Token, winData)
	if err != nil {
		intStatus, err := strconv.Atoi(winResponse.Status)
		if err != nil {
			// handle error
		}
		// Save failed transaction before returning
		go pgs.SaveTransaction(
			session,
			*gc,
			nil,
			"win",
			intStatus,
			winValue,
			99999,
			999+winValue,
		)
		//return -1, winnings, err // Return original API error
		return -1, 0, err // Return original API error
	}
	intStatus, err := strconv.Atoi(winResponse.Status)
	if err != nil {
		// handle error
	}
	go pgs.SaveTransaction(
		session,
		*gc,
		nil,
		"win",
		intStatus,
		winValue,
		99999,
		999+winValue,
	)
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
	//return int(fbalance * 100), winnings, nil
	return int(fbalance * 100), 0, nil
}
