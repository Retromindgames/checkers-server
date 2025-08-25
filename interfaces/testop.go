package interfaces

import (
	"net/http"
	"os"
	"time"

	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

func (m *TestModule) HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, gc *models.GameConfigDTO, rc *redisdb.RedisClient, pgs *postgrescli.PostgresCli) {
	baseUrl := os.Getenv("TEST_OP__GAME_URL")
	session, err := checkExistingSession(req.Token, rc) // TODO: This also needs to check for GameConfig?
	if err != nil || session == nil {
		session, err = generatePlayerSession(req, gc, req.Token, rc)
		if err != nil {
			respondWithError(w, "Failed to generate session", err)
			return
		}

		go handleSaveSession(session, pgs)
	}
	gameURL, err := generateGameURL(baseUrl, req.Token, session.ID, req.Currency)
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

func (m *TestModule) HandleFetchWalletBalance(s models.Session, rc *redisdb.RedisClient) (int64, error) {
	return int64(10000), nil
}

func (m *TestModule) HandlePostBet(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, betValue int64, gameID string) (int64, error) {

	trans := models.Transaction{
		ID:        models.GenerateUUID(),
		SessionID: session.ID,
		Type:      "bet",
		Amount:    betValue,
		Currency:  session.Currency,
		Platform:  "sokkerpro",
		Operator:  "SokkerDuel",
		Client:    session.ClientID,
		//Game:        session.OperatorIdentifier.GameName,
		RoundID:     gameID,
		Timestamp:   time.Now(),
		Status:      "200",
		Description: "Mock transaction",
	}
	go pgs.SaveTransaction(trans)
	return 100, nil
}

func (m *TestModule) HandlePostWin(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, winValue int64, gameID string) (int64, int64, error) {
	trans := models.Transaction{
		ID:        models.GenerateUUID(),
		SessionID: session.ID,
		Type:      "win",
		Amount:    99,
		Currency:  session.Currency,
		Platform:  "sokkerpro",
		Operator:  "SokkerDuel",
		Client:    session.ClientID,
		//Game:        session.OperatorIdentifier.GameName,
		RoundID:     gameID,
		Timestamp:   time.Now(),
		Status:      "200",
		Description: "Mock transaction",
	}
	go pgs.SaveTransaction(trans)
	return 199, 99, nil
}
