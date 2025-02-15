package redisdb

import (
	"checkers-server/models"
	"context"
	"encoding/json"
	"fmt"
)

func (r *RedisClient) AddRoom(key string, room *models.Room) error {
	data, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to serialize room: %v", err)
	}

	exists, err := r.CheckRoomAggregateExists(room.BidAmount)
	if err == nil {
		if exists {
			r.IncrementRoomAggregate(room.BidAmount)
		} else {
			r.CreateRoomAggregate(room.BidAmount)
		}
	}
	return r.Client.HSet(context.Background(), key, room.ID, data).Err()
}

func (r *RedisClient) RoomPlayer(key string) error {
	return r.Client.HDel(context.Background(), key).Err()
}
