package main

import (
	"fmt"
	"net/http"
	"time"
)

// Periodically prints game room status
func monitorGameRooms() {
	for {
		time.Sleep(5 * time.Second) // Adjust as needed

		mutex.Lock()
		roomCount := len(rooms)
		playerCount := 0
		for _, room := range rooms {
			if room.player1 != nil {
				playerCount++
			}
			if room.player2 != nil {
				playerCount++
			}
		}
		mutex.Unlock()

		fmt.Printf("Active Rooms: %d | Total Players: %d\n", roomCount, playerCount)
	}
}

func main() {
	go monitorGameRooms()

	http.HandleFunc("/ws", handleConnection)
	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}