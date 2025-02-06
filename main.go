package main

import (
	"checkers-server/game"
	"checkers-server/handlers"
	"fmt"
	"net/http"
	"time"
)

// Periodically prints game room status
func monitorGameRooms() {
	for {
		time.Sleep(5 * time.Second) // Adjust as needed

		handlers.Mutex.Lock()
		roomCount := len(game.Rooms)
		playerCount := 0
		for _, room := range game.Rooms {
			if room.Player1 != nil {
				playerCount++
			}
			if room.Player2 != nil {
				playerCount++
			}
		}
		handlers.Mutex.Unlock()

		fmt.Printf("Active Rooms: %d | Total Players: %d\n", roomCount, playerCount)
	}
}

func main() {
	go monitorGameRooms()

	http.HandleFunc("/ws", handlers.HandleConnection)
	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}