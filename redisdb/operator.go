package redisdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Lavizord/checkers-server/models"
	"github.com/redis/go-redis/v9"
)

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

func (r *RedisClient) GetAllOperators() ([]models.Operator, error) {
	var operators []models.Operator
	ctx := context.Background()

	err := r.ClusterClient.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
		var cursor uint64
		for {
			keys, nextCursor, err := client.Scan(ctx, cursor, "operator:*", 100).Result()
			if err != nil {
				return fmt.Errorf("redis SCAN failed: %w", err)
			}

			for _, key := range keys {
				val, err := client.Get(ctx, key).Result()
				if err != nil {
					return fmt.Errorf("redis GET failed for key %s: %w", key, err)
				}

				var operator models.Operator
				if err := json.Unmarshal([]byte(val), &operator); err != nil {
					return fmt.Errorf("failed to unmarshal operator for key %s: %w", key, err)
				}
				operators = append(operators, operator)
			}

			if nextCursor == 0 {
				break
			}
			cursor = nextCursor
		}
		return fmt.Errorf("no operators found in redis.")
	})

	if err != nil {
		return nil, err
	}

	return operators, nil
}
