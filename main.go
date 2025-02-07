package main

// TODO: (1) - Implement Auth.
// TODO: (2) - Think about  Redis integraton.

import (
	"checkers-server/core"
	"checkers-server/handlers"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// TODO: Pass this to a consoleUtils package.
// ! Not working right.
// clearConsole clears the console screen.
func clearConsole() {
	switch runtime.GOOS {
	case "linux", "darwin":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "windows": 
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		fmt.Println("\n\n\n\n\n\n\n\n\n\n")
	}
}

func monitorGameRooms() {
	for {
		time.Sleep(2 * time.Second)
		//clearConsole()
		
		onlinePlayers := len(core.ConnectedPlayers)
		queuePlayers := len(core.WaitingQueue)
		activeRooms := len(core.Rooms)
		
		// Count stuff to print it grouped later.
		selectedBidCount := make(map[float64]int)
		for _, player := range core.WaitingQueue {
			selectedBidCount[player.SelectedBid]++
		}
		bidAmountCount := make(map[float64]int)
		for _, room := range core.Rooms {
			bidAmountCount[room.BidAmount]++
		}

		fmt.Printf("Online Players: [%d] | In Queue: [%d] | Active Rooms: [%d]\n",
			onlinePlayers, queuePlayers, activeRooms)
		// Print counted stuff grouped.
		for selectedBid, count := range selectedBidCount {
			fmt.Printf("[%d] Queued Players of SelectedBid [%.2f]\n", count, selectedBid)
		}
		for bidAmount, count := range bidAmountCount {
			fmt.Printf("[%d] - Rooms of BidAmount [%.2f]\n", count, bidAmount)
		}
		fmt.Println("----------------------------------------")
	}
}


func main() {
	go monitorGameRooms()

	http.HandleFunc("/ws", handlers.HandleConnection)
	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}
