package redisdb

import (
	"checkers-server/models"
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisClient represents a wrapper around Redis client.
type RedisClient struct {
	Client *redis.Client
	Ctx    context.Context
}

func NewRedisClient(addr string) (*RedisClient, error) {
	// Initialize the Redis client
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Attempt to ping Redis to check connection
	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("[pkg/redisdb/cliente] - failed to connect to Redis at %s: %w", addr, err)
	}

	// Return the initialized RedisClient on success
	return &RedisClient{Client: client}, nil
}


// RPush - Push serialized player to Redis
func (r *RedisClient) RPush(queue string, player *models.Player) error {
	data, err := json.Marshal(player)
	if err != nil {
		return err
	}
	return r.Client.RPush(context.Background(), queue, string(data)).Err()
}

// BLPop - Retrieve a player from Redis queue
func (r *RedisClient) BLPop(queue string, timeout int) (*models.Player, error) {
	result, err := r.Client.BLPop(context.Background(), 0, queue).Result()
	if err != nil {
		return nil, err
	}

	if len(result) > 1 {
		var player models.Player
		err = json.Unmarshal([]byte(result[1]), &player)
		if err != nil {
			return nil, err
		}
		return &player, nil
	}

	return nil, fmt.Errorf("no player found in queue")
}


func (rc *RedisClient) PublishPlayerEvent(player *models.Player, chanel string) error {
	event := map[string]interface{}{
		"ID":        player.ID,
		"Token":     player.Token,
		"SessionID": player.SessionID,
		"Currency":  player.Currency,
		"status":    player.Status,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("[pkg/redisdb/cliente] - failed to marshal event data: %w", err)
	}

	err = rc.Client.Publish(context.Background(), chanel, data).Err()
	if err != nil {
		return fmt.Errorf("[pkg/redisdb/cliente] - failed to publish player event: %w", err)
	}
	return nil
}

// Subscribe to a Redis Pub/Sub channel
func (r *RedisClient) SubscribePlayerChannel(player models.Player, messageHandler func(string)) {
	pubsub := r.Client.Subscribe(context.Background(), GetPlayerPubSubChannel(player))

	// Start listening for messages
	go func() {
		for msg := range pubsub.Channel() {
			messageHandler(msg.Payload) // Pass message to handler
		}
	}()
}

// Publish a message to a player's Redis Pub/Sub channel , generates chanel from player.
func (r *RedisClient) PublishToPlayer(player models.Player, message string) error {
	return r.Client.Publish(context.Background(), GetPlayerPubSubChannel(player), message).Err()
}

func (r *RedisClient) AddPlayer(key string, player *models.Player) error {
	data, err := json.Marshal(player)
	if err != nil {
		return fmt.Errorf("failed to serialize player: %v", err)
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
		return nil, fmt.Errorf("failed to deserialize player: %v", err)
	}

	return &player, nil
}

func (r *RedisClient) RemovePlayer(key string, playerID string) error {
	return r.Client.HDel(context.Background(), key, playerID).Err()
}