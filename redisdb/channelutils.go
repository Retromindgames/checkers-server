package redisdb

import "checkers-server/models"

func GetPlayerPubSubChannel(player models.Player) (string) {
	return "player:"+string(player.ID)
}
func GenerateRoomRedisKey(room models.Room) (string) {
	return "room:"+string(room.ID)
}