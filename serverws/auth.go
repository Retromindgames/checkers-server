package main

import (
	"fmt"
	"net/http"

	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/platforminterfaces"
	"github.com/Lavizord/checkers-server/redisdb"
)

// Method to validate that the request is valid
func AuthValid(w http.ResponseWriter, r *http.Request, redis *redisdb.RedisClient) (bool, *models.Session, error) {
	token := r.URL.Query().Get("token")
	sessionID := r.URL.Query().Get("sessionid")
	currency := r.URL.Query().Get("currency")
	logger.Default.Infof("[wsapi] HandleConnection: token[%v], sessionid[%v], currency[%v]", token, sessionID, currency)

	session, err := fetchAndValidateSession(token, sessionID, currency, redis)
	if err != nil {
		logger.Default.Warnf("[wsapi] Unauthorized: token[%v], sessionid[%v], currency[%v]: %v ", token, sessionID, currency, err)
		http.Error(w, fmt.Sprintf("Unauthorized: token[%v], sessionid[%v], currency[%v]", token, sessionID, currency), http.StatusUnauthorized)
		if session != nil {
			redis.RemoveSession(session.ID)
		}
		return false, session, err
	}
	return true, session, nil
}

func fetchAndValidateSession(token, sessionID, currency string, redis *redisdb.RedisClient) (*models.Session, error) {
	session, err := redis.GetSessionByID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch session from Redis for sessionID: %v, with err: %v", sessionID, err)
	}

	if session.Currency != currency {
		return nil, fmt.Errorf("currency mismatch for session: %v ,expected %s, got %s", sessionID, currency, session.Currency)
	}

	if session.Token != token {
		return nil, fmt.Errorf("token mismatch for session: %v expected %s, got %s", sessionID, token, session.Token)
	}
	if session.Demo {
		return session, nil
	}
	if session.IsTokenExpired() {
		return nil, fmt.Errorf("token expired for session: %v", sessionID)
	}
	return session, nil
}

func FetchWalletBallance(session *models.Session, redis *redisdb.RedisClient) (int64, error) {
	module := platforminterfaces.PlatformModules[session..OperatorName]
	walletBalance, err := module.HandleFetchWalletBalance(*session, redis)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch wallet for session: %v, with error: %v", session.ID, err)
	}
	return walletBalance, nil
}

func CreatePlayer(redis *redisdb.RedisClient, session *models.Session) (*models.Player, bool, bool, error) {
	var player *models.Player
	wasdisconnected := false

	discPlayer := redis.GetDisconnectedPlayerData(session.ID)
	if discPlayer != nil {
		wasdisconnected = true
		player = &models.Player{
			ID:                 discPlayer.ID,
			Token:              discPlayer.Token,
			Name:               discPlayer.Name,
			SelectedBet:        discPlayer.SelectedBet,
			SessionID:          discPlayer.SessionID,
			Currency:           session.Currency,
			Status:             models.StatusInGame,
			GameID:             discPlayer.GameID,
			OperatorIdentifier: session.OperatorIdentifier,
		}
		redis.AddPlayer(player)
		return player, wasdisconnected, false, nil
	}

	discPlayer = redis.GetDisconnectedInQueuePlayerData(session.ID)
	if discPlayer != nil {
		wasdisconnected = true
		player = &models.Player{
			ID:                 discPlayer.ID,
			Token:              discPlayer.Token,
			Name:               discPlayer.Name,
			SelectedBet:        discPlayer.SelectedBet,
			SessionID:          discPlayer.SessionID,
			Currency:           session.Currency,
			Status:             discPlayer.Status,
			RoomID:             discPlayer.RoomID,
			GameID:             "",
			OperatorIdentifier: session.OperatorIdentifier,
		}
		redis.AddPlayer(player)
		return player, wasdisconnected, true, nil
	}

	existingPlayer, _ := redis.GetPlayer(session.ID)
	if existingPlayer != nil {
		logger.Default.Warnf("Session with id: %v with active player", session.ID)
		return nil, wasdisconnected, false, fmt.Errorf("session with active player")
	}

	newPlayer := &models.Player{
		ID:                 session.ID,
		Token:              session.Token,
		Name:               session.PlayerName,
		SessionID:          session.ID,
		Currency:           session.Currency,
		Status:             models.StatusOnline,
		OperatorIdentifier: session.OperatorIdentifier,
	}
	redis.AddPlayer(newPlayer)
	return newPlayer, wasdisconnected, false, nil
}
