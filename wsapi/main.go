package main

import (
	"checkers-server/wsapi/server"
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/ws", server.HandleConnection)
	fmt.Println("[Player-Connection-Service] - WebSocket server started on :8080")
	http.ListenAndServe(":8080", nil)
}
