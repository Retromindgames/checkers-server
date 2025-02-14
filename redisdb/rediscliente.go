package redisdb

import (
	"checkers-server/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
	Ctx    context.Context
}

func NewRedisClient(addr string) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// check connection
	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("[RedisClient] - failed to connect to Redis at %s: %w", addr, err)
	}
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

func (rc *RedisClient) Publish(channel string, message string) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to marshal message data: %w", err)
	}

	err = rc.Client.Publish(context.Background(), channel, data).Err()
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to publish message: %w", err)
	}
	return nil
}

func (r *RedisClient) SubscribePlayerChannel(player models.Player, messageHandler func(string)) {
	pubsub := r.Client.Subscribe(context.Background(), GetPlayerPubSubChannel(player))

	go func() {
		for msg := range pubsub.Channel() {
			messageHandler(msg.Payload) // Pass message to handler
		}
	}()
}

func (r *RedisClient) Subscribe(channel string, messageHandler func(string)) {
	pubsub := r.Client.Subscribe(context.Background(), channel)

	go func() {
		for msg := range pubsub.Channel() {
			messageHandler(msg.Payload) // Pass message to handler
		}
	}()
}

func (r *RedisClient) PublishToPlayer(player models.Player, message string) error {
	return r.Client.Publish(context.Background(), GetPlayerPubSubChannel(player), message).Err()
}

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

func (r *RedisClient) AddRoom(key string, room *models.Room) error {
	data, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to serialize room: %v", err)
	}

	// Store the room using the roomID as the key
	return r.Client.HSet(context.Background(), key, room.ID, data).Err()
}

func GetRoomAggregates(client *redis.Client) (map[string]*models.RoomAgregate, error) {
	ctx := context.Background()
	roomAggregates := make(map[string]*models.RoomAgregate)

	var cursor uint64
	for {
		// Scan for keys matching "room:*"
		keys, newCursor, err := client.Scan(ctx, cursor, "room:*", 100).Result()
		if err != nil {
			return nil, err
		}
		cursor = newCursor
		for _, key := range keys {
			// Check if IsRoomOpen exists
			exists, err := client.HExists(ctx, key, "IsRoomOpen").Result()
			if err != nil || !exists {
				continue // Skip if the field doesn't exist
			}
			// Get the IsRoomOpen field
			isRoomOpen, err := client.HGet(ctx, key, "IsRoomOpen").Result()
			if err != nil {
				log.Printf("Error retrieving IsRoomOpen for %s: %v", key, err)
				continue
			}
			// Convert IsRoomOpen to bool
			if isRoomOpen != "true" {
				continue
			}
			// Fetch required fields
			bidAmountStr, _ := client.HGet(ctx, key, "BidAmount").Result()
			currency, _ := client.HGet(ctx, key, "Currency").Result()
			gameStatus, _ := client.HGet(ctx, key, "GameStatus").Result()

			bidAmount, err := strconv.ParseFloat(bidAmountStr, 64)
			if err != nil {
				log.Printf("Error parsing BidAmount for %s: %v", key, err)
				continue
			}
			// Convert bidAmount to string for JSON compatibility
			bidAmountKey := strconv.FormatFloat(bidAmount, 'f', -1, 64)
			// Aggregate by BidAmount
			if _, exists := roomAggregates[bidAmountKey]; !exists {
				roomAggregates[bidAmountKey] = &models.RoomAgregate{
					Currency:   currency,
					BidAmount:  bidAmount,
					GameStatus: gameStatus,
				}
			}
		}
		// If cursor is 0, we have scanned all keys
		if cursor == 0 {
			break
		}
	}

	return roomAggregates, nil
}