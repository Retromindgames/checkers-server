package redisdb

import (
	"checkers-server/models"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (r *RedisClient) AddGame(game *models.Game) error {
	data, err := json.Marshal(game)
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to serialize game: %v", err)
	}
	// Uses a shared key ("games") , store the games data under their ID
	return r.Client.HSet(context.Background(), "games", game.ID, data).Err()
}

func (r *RedisClient) UpdateGame(game *models.Game) error {
	exists, err := r.GameExists(game.ID)
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to check game existence: %v", err)
	}
	if !exists {
		return fmt.Errorf("[RedisClient] - game with ID %s does not exist", game.ID)
	}

	data, err := json.Marshal(game)
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to serialize game: %v", err)
	}

	return r.Client.HSet(context.Background(), "games", game.ID, data).Err()
}

func (r *RedisClient) GetGame(gameID string) (*models.Game, error) {
	data, err := r.Client.HGet(context.Background(), "games", gameID).Result()
	if err != nil {
		return nil, fmt.Errorf("[RedisClient] - failed to get game: %v", err)
	}

	var game models.Game
	err = json.Unmarshal([]byte(data), &game)
	if err != nil {
		return nil, fmt.Errorf("[RedisClient] - failed to deserialize Game: %v", err)
	}
	return &game, nil
}

func (r *RedisClient) RemoveGame(gameID string) error {
	exists, err := r.GameExists(gameID)
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to check if game exists: %v", err)
	}
	if !exists {
		return fmt.Errorf("[RedisClient] - atempting to delete game that does not exist: %s", gameID)
	}
	return r.Client.HDel(context.Background(), "games", gameID).Err()
}

func (r *RedisClient) GameExists(gameID string) (bool, error) {
	exists, err := r.Client.HExists(context.Background(), "games", gameID).Result()
	return exists, err
}

func (r *RedisClient) GetNumberOfGames() int {
	// Use HLen to get the number of fields (games) in the "games" hash
	count, err := r.Client.HLen(context.Background(), "games").Result()
	if err != nil {
		fmt.Errorf("[RedisClient] - failed to get number of games: %v", err)
		return 0
	}
	return int(count)
}

// This should be called when a disconnect happens during a game, it will save the
// session and some player data, to easily identify disconnected players.
func (r *RedisClient) SaveDisconnectSessionPlayerData(playerData models.Player, game models.Game) {
	if playerData.DisconnectedAt == 0 { // Check if it's unset
		playerData.DisconnectedAt = time.Now().Unix() // Set current timestamp
	}

	playerJSON, err := json.Marshal(playerData)
	if err != nil {
		fmt.Println("Error marshaling player:", err)
		return
	}
	key := fmt.Sprintf("players_disconnected:%s", playerData.SessionID)

	err = r.Client.Set(context.Background(), key, playerJSON, 0).Err()
	if err != nil {
		fmt.Println("Error saving player to Redis:", err)
		return
	}
	fmt.Println("Player saved to disconnected list with key:", key)
}

// This retrieves our player disconnect, should be used to check if the player that just logged in is in a match.
func (r *RedisClient) GetDisconnectedPlayerData(sessionID string) *models.Player {
	key := fmt.Sprintf("players_disconnected:%s", sessionID)

	// Get the JSON data from Redis
	playerJSON, err := r.Client.Get(context.Background(), key).Result()
	if err != nil {
		fmt.Println("Error retrieving player from Redis:", err)
		return nil
	}

	// Unmarshal the JSON data into a GamePlayer struct
	var player models.Player
	err = json.Unmarshal([]byte(playerJSON), &player)
	if err != nil {
		fmt.Println("Error unmarshaling player JSON:", err)
		return nil
	}

	return &player
}

func (r *RedisClient) DeleteDisconnectedPlayerSession(sessionID string) error {
	key := fmt.Sprintf("players_disconnected:%s", sessionID)

	// Delete the key from Redis
	err := r.Client.Del(context.Background(), key).Err()
	if err != nil {
		fmt.Println("Error deleting player session from Redis:", err)
		return err
	}

	fmt.Println("Player session deleted with key:", key)
	return nil
}
