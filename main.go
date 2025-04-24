package main

import (
	"checkers-server/config"
	"checkers-server/redisdb"
	"checkers-server/wsapi"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {

	config.LoadConfig()
	ports := config.Cfg.Services["wsapi"].Ports
	if len(ports) == 0 {
		log.Fatal("[wsapi] - No ports defined for wsapi\n")
	}
	redisConData := config.Cfg.Redis
	client, err := redisdb.NewRedisClient(redisConData.Addr, redisConData.User, redisConData.Password)
	if err != nil {
		log.Panicf("[Redis] Error initializing Redis client: %v", err)
	}
	wsapi.RedisClient = client

	// Get SSL cert paths from env
	certPath := os.Getenv("SSL_CERT_PATH")
	keyPath := os.Getenv("SSL_KEY_PATH")

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go wsapi.SubscribeToBroadcastChannel() // This is a global channel. WSAPI will send the messages from this channel to all active ws connections

	if certPath == "" || keyPath == "" {
		port := ports[0] // First port for HTTP
		addr := fmt.Sprintf(":%d", port)
		http.HandleFunc("/ws", wsapi.HandleConnection)
		log.Println("[wsapi] - SSL certificate paths not set, defaulting to listen on HTTP.")
		log.Printf("[wsapi] - WebSocket server started on %s\n", addr)
		log.Fatal(http.ListenAndServe(addr, nil))
	} else {
		port := ports[1] // Second port for SSL
		addr := fmt.Sprintf(":%d", port)
		http.HandleFunc("/ws", wsapi.HandleConnection)
		log.Println("[wsapi] - SSL certificate paths set, listening on HTTPS .")
		log.Printf("[wsapi] - WebSocket server started on %s\n", addr)
		log.Fatal(http.ListenAndServeTLS(addr, certPath, keyPath, nil))
	}
}
