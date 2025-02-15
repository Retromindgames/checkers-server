package redisdb

import (
	"checkers-server/models"
	"context"
	"encoding/json"
	"fmt"
)

func (r *RedisClient) AddPlayer(key string, player *models.Player) error {
	data, err := json.Marshal(player)
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to serialize player: %v", err)
	}

	return r.Client.HSet(context.Background(), key, player.ID, data).Err()
}

func (r *RedisClient) GetPlayer(key string, playerID string) (*models.Player, error) {
	data, err := r.Client.HGet(context.Background(), key, playerID).Result()
	if err != nil {
		return nil, err
	}

	var player models.Player
	err = json.Unmarshal([]byte(data), &player)
	if err != nil {
		return nil, fmt.Errorf("[RedisClient] - failed to deserialize player: %v", err)
	}

	return &player, nil
}

func (r *RedisClient) RemovePlayer(key string, playerID string) error {
	return r.Client.HDel(context.Background(), key, playerID).Err()
}
