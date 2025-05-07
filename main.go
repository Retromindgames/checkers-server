package main

import (
	"checkers-server/config"
	"checkers-server/wsapi"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	defer func() {
		if wsapi.RedisClient != nil {
			wsapi.RedisClient.CloseRedisClient()
		}
	}()

	config.LoadConfig()
	ports := config.Cfg.Services["wsapi"].Ports
	if len(ports) == 0 {
		log.Fatal("[wsapi] - No ports defined for wsapi\n")
	}

	// Get SSL cert paths from env
	certPath := os.Getenv("SSL_CERT_PATH")
	keyPath := os.Getenv("SSL_KEY_PATH")

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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
