package main

import (
	"context"
	"fmt"

	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/redisdb"
	"github.com/redis/go-redis/v9"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	redis *redisdb.RedisClient

	broadastpubsub *redis.PubSub

	gameName string

	//pubsub *redisdb.RedisClient.PubSub

}

func newHub(addr string, username string, password string, redistls bool, gn string) *Hub {
	redisclient, err := redisdb.NewRedisClient(addr, username, password, redistls)
	if err != nil {
		logger.Default.Fatalf("[Redis] Error initializing Redis client: %v", err)
	}
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		redis:      redisclient,
		gameName:   gn,
	}
}

func (h *Hub) Close() {
	if h.broadastpubsub != nil {
		h.broadastpubsub.Close()
	}
	h.redis.CloseRedisClient()
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			logger.Default.Infof("[HUB.Run] - Unregister signal for session: %v", client.player.ID)
			if _, ok := h.clients[client]; ok {
				h.CloseConnection(client)
			}

		case message := <-h.broadcast:
			//msg, _ := json.Marshal(message) //log.Printf("[HUB.Run] - Broadcast: %v", msg)
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					h.CloseConnection(client)
				}
			}
		}
	}
}

func (h *Hub) CloseConnection(client *Client) {
	logger.Default.Infof("[HUB.CloseConnection] - closing connection for session: %v", client.player.ID)
	client.UpdatePlayerDataFromRedis()

	// If our player is in queue, we add it to a special redis set to handle players in queue that were disconnected.
	// Added here the status in room, i think it will share the same logic.
	if client.player.Status == models.StatusInQueue || client.player.Status == models.StatusInRoom || client.player.Status == models.StatusInRoomReady {
		logger.Default.Infof("[HUB.CloseConnection] - Removed player is in a Room, saving player to in queue disconnect with session: %v", client.player.ID)
		h.redis.SaveDisconnectInQueuePlayerData(client.player)
	}
	if client.player.RoomID != "" || client.player.Status == models.StatusInRoom || client.player.Status == models.StatusInRoomReady {
		// log.Printf("[Hub.Run] - Removed player is in a Room, sending notification to room worker!: %v\n", client.player)
		//h.redis.RPush("leave_room", client.player)
		//h.redis.PublishToRoomPubSub(client.player.RoomID, "leave_room:"+client.player.ID)
	}
	if client.player.GameID != "" || client.player.Status == models.StatusInGame {
		logger.Default.Infof("[Hub.CloseConnection] - Removed player is in a Game, sending notification to Game worker for session: %v", client.player.ID)
		listName := fmt.Sprintf("disconnect_game:{%v}", h.gameName)
		h.redis.RPush(listName, client.player)
	}
	h.redis.RemovePlayer(client.player.ID)
	delete(h.clients, client)
	close(client.send)
}

func (h *Hub) SubscribeBroadcast() {
	pubsub := h.redis.Client.Subscribe(context.Background(), fmt.Sprintf("game_info:{%v}", h.gameName))
	h.broadastpubsub = pubsub
	ch := pubsub.Channel()

	go func() {
		for msg := range ch {
			//log.Printf("[HUB.SubscribeBroadcast]: %v", msg.Payload)
			h.broadcast <- []byte(msg.Payload) // Send to all connected clients
		}
	}()
}
