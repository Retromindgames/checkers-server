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
	"sync"

	"github.com/gorilla/websocket"
)

var redisClient *redisdb.RedisClient

var (
	players      = make(map[string]*models.Player) // Store players by ID
	playersMutex = sync.Mutex{}                    // Protects the map
)

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
	go subscribeToBroadcastChannel() // This is a global channel. WSAPI will send the messages
	// from this channel to all active ws connections
}

func HandleConnection(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	sessionID := r.URL.Query().Get("sessionid")
	currency := r.URL.Query().Get("currency")

	valid, playerData := IsUserValid(token, sessionID) // TODO: Have this check against a valid DB
	if !valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to upgrade:", err)
		conn.Close()
		return
	}
	var player *models.Player
	// We will just check if the player that connected is part of our disconnected players
	discPlayer := redisClient.GetDisconnectedPlayerData(sessionID)
	if discPlayer != nil {
		player = &models.Player{
			ID:             discPlayer.ID,
			Conn:           conn,
			Token:          playerData.Token,
			Name:           playerData.Name,
			SessionID:      playerData.SessionID,
			Currency:       currency,
			CurrencyAmount: playerData.CurrencyAmount,
			Status:         models.StatusInGame,
			GameID:         discPlayer.GameID,
			WriteChan:      make(chan []byte),
		}
	} else {
		playerID := models.GenerateUUID()
		newPlayer := &models.Player{
			ID:             playerID,
			Conn:           conn,
			Token:          playerData.Token,
			Name:           playerData.Name,
			SessionID:      playerData.SessionID,
			Currency:       currency,
			CurrencyAmount: playerData.CurrencyAmount,
			Status:         models.StatusOnline,
			WriteChan:      make(chan []byte),
		}
		player = newPlayer
	}
	player.StartWriteGoroutine() // Start the write goroutine

	// We add the player to our player map.
	playersMutex.Lock()
	players[player.ID] = player
	playersMutex.Unlock()

	subscriptionReady := make(chan bool)
	go subscribeToPlayerChannel(player, subscriptionReady)
	<-subscriptionReady // Wait for the subscription to be ready

	err = redisClient.RPush("player_online", player)
	if err != nil {
		fmt.Println("[wsapi] - Failed to push player online", err)
		playersMutex.Lock()
		delete(players, player.ID)
		playersMutex.Unlock()
		unsubscribeFromPlayerChannel(player)
		conn.Close()
		return
	}
	fmt.Println("[wsapi] - Player added online:", player.ID)
	// Now that our player has subscribbed to our stuff, we will notify the gameworker of the reconnect.
	if discPlayer != nil {
		redisClient.RPush("reconnect_game", player)
	}
	go handleMessages(player)
}

// Function to handle player channel subscription
func subscribeToPlayerChannel(player *models.Player, ready chan bool) {
	redisClient.SubscribePlayerChannel(*player, func(message string) {
		fmt.Println("[wsapi] - Received server to PLAYER message:", message)
		// Send the received message to the player's WebSocket connection
		player.WriteChan <- []byte(message)
	})
	ready <- true // Notify that the subscription is ready
}


func subscribeToBroadcastChannel() {
	redisClient.Subscribe("game_info", func(message string) {
		fmt.Println("[wsapi] - Received BROADCAST message:", message)
		// Step 1: Parse the message using messages.ParseMessage
		msg, err := messages.ParseMessage([]byte(message))
		if err != nil {
			fmt.Println("[wsapi] - Failed to parse message:", err)
			return
		}
		// Step 2: Marshal the message back to JSON
		finalBytes, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("[wsapi] - Failed to marshal final message:", err)
			return
		}
		// Step 3: Send the message to all connected players
		playersMutex.Lock()
		defer playersMutex.Unlock() // Ensures mutex is unlocked even if an error occurs
		for _, player := range players {
			player.WriteChan <- finalBytes // Send message to the write channel
		}
	})
}

func unsubscribeFromPlayerChannel(player *models.Player) {
	redisClient.UnsubscribePlayerChannel(*player)
}

func unsubscribeFromBroadcastChannel(player *models.Player) {
	redisClient.Unsubscribe("game_info")
}

// Mock user validation
func IsUserValid(token string, sessionID string) (bool, models.Player) {
	player, exists := redisdb.MockPlayers[token]
	if exists && player.SessionID == sessionID {
		return true, player
	}
	return false, models.Player{} // Invalid user
}
