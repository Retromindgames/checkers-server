package server

import (
	"checkers-server/pkg/redisdb"
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
	} else {
		fmt.Println("[Redis] client initialized.")
	}
	redisClient = client
}

// -> ws://localhost:8080/ws?token=abc123&sessionid=xyz789&currency=USD
func HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Extract token, sessionid, and currency from query parameters
	token := r.URL.Query().Get("token")
	sessionID := r.URL.Query().Get("sessionid")
	currency := r.URL.Query().Get("currency")

	// Upgrade the HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to upgrade:", err)
		return
	}

	player := &redisdb.Player{
		ID:       r.RemoteAddr, // TODO: Generate a proper player ID?
		Conn:     conn,
		Token:    token,
		SessionID: sessionID,
		Currency: currency,
		Status: "connected",
	}
	
	// Log the new player connection details
	fmt.Println("New player connected:", player.ID)
	// Publish the connection event to Redis
	err = redisClient.PublishPlayerEvent(player, "player-connected")
	if err != nil {
		fmt.Println("Failed to publish player event:", err)
		return
	}
	go handleMessages(player)
}


