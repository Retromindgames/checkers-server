package server

import (
	"checkers-server/models"
	"checkers-server/redisdb"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var redisClient *redisdb.RedisClient

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func init() {
	client, err := redisdb.NewRedisClient("localhost:6379")
	if err != nil {
		log.Fatalf("[Redis] Error initializing Redis client: %v", err)
	}
	redisClient = client
}

func HandleConnection(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	sessionID := r.URL.Query().Get("sessionid")
	currency := r.URL.Query().Get("currency")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to upgrade:", err)
		return
	}

	playerID := r.RemoteAddr // TODO: Generate a proper player ID
	player := &models.Player{
		ID:        playerID,
		Conn:      conn,
		Token:     token,
		SessionID: sessionID,
		Currency:  currency,
		Status:    "connected",
	}

	// RPush, seends a message to be processed once by a worker service.
	err = redisClient.RPush("player_online", player)
	if err != nil {
		fmt.Println("[wsapi] - Failed to push player online", err)
		return
	}
	fmt.Println("[wsapi] - Player added to queue:", player.ID)

	// Start processing the queue (blocking pop)
	go handleMessages(player);
}

