package server

import (
	"checkers-server/models"
	"checkers-server/redisdb"
	"fmt"
	"log"
	"net/http"
	"sync"

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
	playerID := r.RemoteAddr 
	player := &models.Player{
		ID:        playerID,
		Conn:      conn,
		Token:     token,
		SessionID: sessionID,
		Currency:  currency,
		Status:    "connected",
	}
	
	// TODO: This seemed to solve an issue, make sure it is needed...
	// Use a WaitGroup to wait for the subscription to finish before proceeding
	var wg sync.WaitGroup
	wg.Add(1) // Wait for the subscription to be set up
	// Subscribe to the player's Redis channel in a separate goroutine
	go func() {
		defer wg.Done() // Mark the subscription as done once it starts
		subscribeToPlayerChannel(player)
	}() // Wait for the subscription to be ready before proceeding with other actions
	wg.Wait()	// seends a message to be processed once by a worker.
	
	err = redisClient.RPush("player_online", player)
	if err != nil {
		fmt.Println("[wsapi] - Failed to push player online", err)
		return
	}
	fmt.Println("[wsapi] - Player added online:", player.ID)
	go handleMessages(player);
}

// ssubscribes to a Redis channel / sends messages to the WebSocket 
func subscribeToPlayerChannel(player *models.Player) {
	go redisClient.SubscribePlayerChannel(*player, func(message string) {
		err := player.Conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			fmt.Println("[wsapi] - Failed to send message to player:", err)
		}
	})
}
