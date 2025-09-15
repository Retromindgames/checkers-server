package redisdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli/ent"
	"github.com/redis/go-redis/v9"
)

// TODO: Refractor to be inline with AddOperators()
func (r *RedisClient) AddOperator(operator *models.Operator) error {
	ctx := context.Background()
	ttl := 24 * time.Hour

	key := fmt.Sprintf("operator:%s:%s", operator.OperatorName, operator.OperatorGameName)

	data, err := json.Marshal(operator)
	if err != nil {
		return fmt.Errorf("[RedisClient] - AddOperator failed to serialize operator: %v", err)
	}

	log.Printf("Adding operator %s to Redis\n", key)
	return r.Client.Set(ctx, key, data, ttl).Err()
}

func (r *RedisClient) AddOperators(operators []*ent.Operator) error {
	ctx := context.Background()
	ttl := 24 * time.Hour

	for _, op := range operators {
		dto := models.OperatorToDTO(op)

		key := fmt.Sprintf("operator:%s:%s", op.Name, op.Alias)
		jsonData, err := json.Marshal(dto)
		if err != nil {
			return fmt.Errorf("marshal operator %s: %w", op.Name, err)
		}

		if err := r.Client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
			return fmt.Errorf("cache operator %s: %w", op.Name, err)
		}

		// Track in set for fast listing
		if err := r.Client.SAdd(ctx, "operators:set", key).Err(); err != nil {
			return fmt.Errorf("track operator key: %w", err)
		}
	}
	return nil
}

func (r *RedisClient) GetAllOperators() ([]models.OperatorDTO, error) {
	ctx := context.Background()

	keys, err := r.Client.SMembers(ctx, "operators:set").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch operator keys: %w", err)
	}
	if len(keys) == 0 {
		return nil, nil
	}

	results, err := r.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch operators: %w", err)
	}

	operators := make([]models.OperatorDTO, 0, len(results))
	for _, res := range results {
		if res == nil {
			continue
		}
		var dto models.OperatorDTO
		if err := json.Unmarshal([]byte(res.(string)), &dto); err != nil {
			return nil, fmt.Errorf("failed to unmarshal operator: %w", err)
		}
		operators = append(operators, dto)
	}

	return operators, nil
}

// TODO: Refractor to be inline with GetAllOperators()
func (r *RedisClient) GetOperator(operatorName, operatorGameName string) (*models.Operator, error) {
	ctx := context.Background()
	key := fmt.Sprintf("operator:%s:%s", operatorName, operatorGameName)

	val, err := r.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("operator not found in Redis: %s", key)
	} else if err != nil {
		return nil, fmt.Errorf("redis GET failed: %w", err)
	}

	var operator models.Operator
	if err := json.Unmarshal([]byte(val), &operator); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operator: %w", err)
	}

	return &operator, nil
}
