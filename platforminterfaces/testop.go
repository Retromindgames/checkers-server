package platforminterfaces

import (
	"net/http"
	"os"

	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

func (m *TestModule) HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, gc *models.GameConfigDTO, rc *redisdb.RedisClient, pgs *postgrescli.PostgresCli) {
	baseUrl := os.Getenv("TEST_OP__GAME_URL")

	session, err := checkExistingSession(req.Token, rc)
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

func (m *TestModule) HandleFetchWalletBalance(s models.Session, gc *models.GameConfigDTO, rc *redisdb.RedisClient) (int, error) {
	return int(10000), nil
}

func (m *TestModule) HandlePostBet(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, gc *models.GameConfigDTO, betAmount int, roundID string) (int, error) {
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
	return 100, nil
}

func (m *TestModule) HandlePostWin(pgs *postgrescli.PostgresCli, rc *redisdb.RedisClient, session models.Session, gc *models.GameConfigDTO, winValue int, roundID string) (int, int, error) {
	go pgs.SaveTransaction(
		session,
		*gc,
		nil,
		"bet",
		200,
		winValue,
		99999,
		999+winValue,
	)
	return 199, 99, nil
}
