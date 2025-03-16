package main

import (
	"checkers-server/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// OperatorModule defines the interface for operator-specific code
type OperatorInterface interface {
	HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest)
}

// OperatorModules maps operator names to their respective modules
var OperatorModules = map[string]OperatorInterface{
	"SokkerDuel":      &SokkerDuelModule{},
	"AnotherOperator": &AnotherOperatorModule{},
	// Add more operators here
}

// SokkerDuelModule handles requests for the SokkerDuel operator
type SokkerDuelModule struct{}

func (m *SokkerDuelModule) HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest) {
	// Custom logic for SokkerDuel
	response := map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("SokkerDuel: Game launch request received for game ID: %s", req.GameID),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// AnotherOperatorModule handles requests for another operator
type AnotherOperatorModule struct{}

func (m *AnotherOperatorModule) HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest) {
	// Custom logic for AnotherOperator
	response := map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("AnotherOperator: Game launch request received for game ID: %s", req.GameID),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func gameLaunchHandler(w http.ResponseWriter, r *http.Request) {
	var req models.GameLaunchRequest

	// Decode the JSON body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the request
	if req.GameID == "" || req.OperatorName == "" {
		http.Error(w, "Game ID and Operator Name are required", http.StatusBadRequest)
		return
	}

	// Log the received request
	log.Printf("Received Game Launch Request: %+v", req)

	// Look up the module for the operator
	module, exists := OperatorModules[req.OperatorName]
	if !exists {
		http.Error(w, fmt.Sprintf("Unsupported operator: %s", req.OperatorName), http.StatusBadRequest)
		return
	}

	// Delegate the request to the module
	module.HandleGameLaunch(w, r, req)
}

func main() {
	// Create a new Gorilla mux router
	router := mux.NewRouter()

	// REST API POST endpoint
	router.HandleFunc("/gamelaunch", gameLaunchHandler).Methods("POST")

	// Start the server
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
