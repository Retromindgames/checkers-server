package redisdb

import (
	"checkers-server/pkg/redisdb"
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisSubscriber struct {
	Client *redis.Client
}


func NewRedisSubscriber(addr string) (*RedisSubscriber, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})

	// Test connection
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

	return &RedisSubscriber{Client: client}, nil
}

func (rs *RedisSubscriber) SubscribeToPlayerEvents() {
	SubscribeToConnection(rs)
}

func SubscribeToConnection(rs *RedisSubscriber) {
	pubsub := rs.Client.Subscribe(context.Background(), "player-connected")
	defer pubsub.Close()

	// Channel to receive messages
	ch := pubsub.Channel()

	for msg := range ch {
		// Handle incoming message (event)
		fmt.Printf("Received message: %s\n", msg.Payload)

		// Parse the JSON message into a Player struct
		var player redisdb.Player
		if err := json.Unmarshal([]byte(msg.Payload), &player); err != nil {
			fmt.Printf("Error parsing message: %v\n", err)
			continue
		}
		SavePlayerToRedis(&player, rs.Client)
		// Save the player information to Redis
		//playerKey := fmt.Sprintf("player:%s", player.ID) 
		//// Store the player data as a hash in Redis
		//err := rs.Client.HSet(context.Background(), playerKey, "status", player).Err()
		//if err != nil {
		//	fmt.Printf("Error saving player to Redis: %v\n", err)
		//	continue
		//}

	}
}

func SubscribeToDisconection(rs *RedisSubscriber) {
	pubsub := rs.Client.Subscribe(context.Background(), "player-disconnected")
	defer pubsub.Close()

	// Channel to receive messages
	ch := pubsub.Channel()

	for msg := range ch {
		// Handle incoming message (event)
		fmt.Printf("Received message: %s\n", msg.Payload)

		// Parse the JSON message into a Player struct
		var player redisdb.Player
		if err := json.Unmarshal([]byte(msg.Payload), &player); err != nil {
			fmt.Printf("Error parsing message: %v\n", err)
			continue
		}
		// TODO: Delete player from rediis

	}
}

// TODO: Need to try the two methods bellow.
// SavePlayerToRedis stores player information in Redis as a hash.
func SavePlayerToRedis(player *redisdb.Player, client *redis.Client) error {
	// Create a unique Redis key for the player
	playerKey := fmt.Sprintf("player:%s", player.ID)

	// Store player information as a Redis hash
	err := client.HSet(context.Background(), playerKey, map[string]interface{}{
		"ID":        player.ID,
		"Status":    player.Status,
		"Currency":  player.Currency,
		"SessionID": player.SessionID,
	}).Err()

	if err != nil {
		return fmt.Errorf("error saving player to Redis: %v", err)
	}

	fmt.Println("Player saved to Redis as hash:", player.ID)
	return nil
}

func getPlayerFieldFromRedis(playerID string, field string, client *redis.Client) (string, error) {
    playerKey := fmt.Sprintf("player:%s", playerID)
    fieldValue, err := client.HGet(context.Background(), playerKey, field).Result()
    if err != nil {
        return "", fmt.Errorf("error retrieving player field %s: %v", field, err)
    }
    return fieldValue, nil
}