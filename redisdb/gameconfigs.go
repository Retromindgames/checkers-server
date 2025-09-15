package redisdb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli/ent"
	"github.com/redis/go-redis/v9"
)

func (r *RedisClient) AddGameConfigs(configs []*ent.GameConfig) error {
	for _, gc := range configs {
		err := r.AddGameConfig(gc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RedisClient) AddGameConfig(configs *ent.GameConfig) error {
	ctx := context.Background()
	ttl := 24 * time.Hour

	dto := models.GameConfigToDTO(configs)

	key := fmt.Sprintf("gameconfigs:%s:%s:%s", dto.GameName, dto.OperatorName, dto.CurrencyName)
	jsonData, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshal operator %s: %w", dto.OperatorName, err)
	}

	if err := r.Client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("cache operator %s: %w", dto.OperatorName, err)
	}

	// Track in set for fast listing
	if err := r.Client.SAdd(ctx, "operators:set", key).Err(); err != nil {
		return fmt.Errorf("track operator key: %w", err)
	}
	return nil
}

func (r *RedisClient) GetGameConfig(gameName, operatorName, currencyName string) (*models.GameConfigDTO, error) {
	ctx := context.Background()

	key := fmt.Sprintf("gameconfigs:%s:%s:%s", gameName, operatorName, currencyName)
	val, err := r.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("no game config found for key %s", key)
	} else if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	var dto models.GameConfigDTO
	if err := json.Unmarshal([]byte(val), &dto); err != nil {
		return nil, fmt.Errorf("unmarshal error for key %s: %w", key, err)
	}

	return &dto, nil
}
