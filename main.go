package main

import (
	"checkers-server/core"
	"checkers-server/handlers"
	"fmt"
	"net/http"
	"time"
)

func monitorGameRooms() {
	for {
		time.Sleep(5 * time.Second)

		roomCount := len(core.Rooms)
		playerCount := 0
		for _, room := range core.Rooms {
			if room.Player1 != nil {
				playerCount++
			}
			if room.Player2 != nil {
				playerCount++
			}
		}

		fmt.Printf("Active Rooms: %d | Total Players: %d\n", roomCount, playerCount)
	}
}

func main() {
	go monitorGameRooms()

	http.HandleFunc("/ws", handlers.HandleConnection)
	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}
