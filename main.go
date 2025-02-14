package main

import (
	"checkers-server/config"
	"checkers-server/wsapi"
	"fmt"
	"log"
	"net/http"
)

func main() {
	config.LoadConfig("config/config.json")
	ports := config.Cfg.Services["wsapi"].Ports
	if len(ports) == 0 {
		log.Fatal("[wsapi] - No ports defined for wsapi\n")
	}
	port := ports[0] 				// Select first available port (modify as needed)
	http.HandleFunc("/ws", wsapi.HandleConnection)
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("[wsapi] - WebSocket server started on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
