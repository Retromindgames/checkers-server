package main

import (
	"checkers-server/config"
	"checkers-server/wsapi"
	"fmt"
	"log"
	"net/http"
)

func main() {
	config.LoadConfig()
	ports := config.Cfg.Services["wsapi"].Ports
	if len(ports) == 0 {
		log.Fatal("[wsapi] - No ports defined for wsapi\n")
	}
	port := ports[0] // Select first available port

	
	http.HandleFunc("/ws", wsapi.HandleConnection)
	addr := fmt.Sprintf(":%d", port)
	
	fmt.Printf("[wsapi] - WebSocket server started on %s\n", addr)
	
	// Get SSL cert paths from env
	//certPath := os.Getenv("SSL_CERT_PATH")
	//keyPath := os.Getenv("SSL_KEY_PATH")
	
	//if certPath == "" || keyPath == "" {
		log.Fatal("[wsapi] - SSL certificate paths not set, defaulting to HTTP.")
		log.Fatal(http.ListenAndServe(addr, nil))
	//} else {
	//	log.Fatal(http.ListenAndServeTLS(addr, certPath, keyPath, nil))
	//}	
}
