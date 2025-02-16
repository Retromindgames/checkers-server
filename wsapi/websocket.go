package wsapi

import (
	"checkers-server/config"
	"checkers-server/messages"
	"checkers-server/models"
	"checkers-server/redisdb"
	"encoding/json"
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
	config.LoadConfig()
	redisAddr := config.Cfg.Redis.Addr
	client, err := redisdb.NewRedisClient(redisAddr)
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
		ID:             playerID,
		Conn:           conn,
		Token:          token,
		Name:           "PLACEHOLDER-SERVER",
		SessionID:      sessionID,
		Currency:       currency,
		CurrencyAmount: 999,
		Status:         "connected",
	}

	subscriptionReady := make(chan bool)
	go subscribeToPlayerChannel(player, subscriptionReady)
	<-subscriptionReady // Wait for the subscription to be ready

	subscriptionReady = make(chan bool)
	go subscribeToBroadcastChannel(player, subscriptionReady)
	<-subscriptionReady // Wait for the subscription to be ready

	err = redisClient.RPush("player_online", player)
	if err != nil {
		fmt.Println("[wsapi] - Failed to push player online", err)
		return
	}
	fmt.Println("[wsapi] - Player added online:", player.ID)
	go handleMessages(player)
}

// Function to handle player channel subscription
func subscribeToPlayerChannel(player *models.Player, ready chan bool) {
	redisClient.SubscribePlayerChannel(*player, func(message string) {
		// Send the received message to the player's WebSocket connection
		err := player.Conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			fmt.Println("[wsapi] - Failed to send message to player:", err)
			player.Conn.Close()
		}
	})
	ready <- true // Notify that the subscription is ready
}

func subscribeToBroadcastChannel(player *models.Player, ready chan bool) {
	redisClient.Subscribe("room_info", func(message string) {
		fmt.Println("[wsapi] - broadcast message:", message)
		fmt.Printf("[wsapi] - Type of message: %T\n", message)

		// Step 1: Unmarshal the outer message (it's a string, so we need to parse it)
		var msg messages.Message
		err := json.Unmarshal([]byte(message), &msg)
		if err != nil {
			fmt.Println("[wsapi] - Failed to unmarshal message:\n", err)
			return
		}

		// Step 2: msg.Value is already JSON (since we used json.RawMessage in the producer)
		// Convert it to a clean JSON string
		finalBytes, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("[wsapi] - Failed to marshal final message:\n", err)
			return
		}

		// Step 3: Send the properly formatted JSON to WebSocket
		err = player.Conn.WriteMessage(websocket.TextMessage, finalBytes)
		if err != nil {
			fmt.Println("[wsapi] - Failed to send message to player:\n", err)
			player.Conn.Close()
		}
	})
	ready <- true // Notify that the subscription is ready
}

func unsubscribeFromPlayerChannel(player *models.Player) {
	redisClient.UnsubscribePlayerChannel(*player)
}

func unsubscribeFromBroadcastChannel(player *models.Player) {
	redisClient.Unsubscribe("room_info")
}
