package redisdb

import (
	"checkers-server/models"
	"context"
	"encoding/json"
	"fmt"
)

func (r *RedisClient) AddGame(game *models.Game) error {
	data, err := json.Marshal(game)
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to serialize game: %v", err)
	}
	// Uses a shared key ("games") , store the games data under their ID
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
	exists, err := r.Client.HExists(context.Background(), "game", gameID).Result()
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to check if game exists: %v", err)
	}

	if !exists {
		return fmt.Errorf("[RedisClient] - atempting to delete game that does not exist: %s", gameID)
	}
	return r.Client.HDel(context.Background(), "games", gameID).Err()
}

