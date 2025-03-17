package main

import (
	"checkers-server/config"
	"checkers-server/models"
	"checkers-server/postgrescli"
	"checkers-server/redisdb"
	"checkers-server/walletrequests"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/mux"
)

var pid int
var postgresClient *postgrescli.PostgresCli
var redisClient *redisdb.RedisClient
var name = "restapiworker"

func init() {
	pid = os.Getpid()
	config.LoadConfig()

	redisAddr := config.Cfg.Redis.Addr
	client, err := redisdb.NewRedisClient(redisAddr)
	if err != nil {
		log.Fatalf("[%s-Redis] Error initializing Redis client: %v\n", name, err)
	}
	redisClient = client

	sqlcliente, err := postgrescli.NewPostgresCli(
		config.Cfg.Postgres.User,
		config.Cfg.Postgres.Password,
		config.Cfg.Postgres.DBName,
		config.Cfg.Postgres.Host,
		config.Cfg.Postgres.Port,
	)
	if err != nil {
		log.Fatalf("[%PostgreSQL] Error initializing POSTGRES client: %v\n", err)
	}
	postgresClient = sqlcliente
}

// OperatorModule defines the interface for operator-specific code
type OperatorInterface interface {
	HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, op models.Operator)
}

// OperatorModules maps operator names to their respective modules
var OperatorModules = map[string]OperatorInterface{
	"SokkerDuel": &SokkerDuelModule{},
	//"AnotherOperator": &AnotherOperatorModule{},
	// Add more operators as needed
}

// SokkerDuelModule handles requests for the SokkerDuel operator
type SokkerDuelModule struct{}

func (m *SokkerDuelModule) HandleGameLaunch(w http.ResponseWriter, r *http.Request, req models.GameLaunchRequest, op models.Operator) {
	// Fetch wallet information
	logInResponse, err := walletrequests.SokkerDuelGetWallet(op, req.Token)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch wallet: %v", err), http.StatusInternalServerError)
		return
	}

	session, err := generatePlayerSession(
		op,
		req.Token,
		logInResponse.Data.Username,
		logInResponse.Data.Currency,
		logInResponse.Data.Balance,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate session: %v", err), http.StatusInternalServerError)
		return
	}

	gameURL, err := generateGameURL(op.GameBaseUrl, req.Token, session.ID, logInResponse.Data.Currency)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate game URL: %v", err), http.StatusInternalServerError)
		return
	}

	response := models.SokkerDuelGamelaunchResponse{
		Token: req.Token,
		Url:   gameURL,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func gameLaunchHandler(w http.ResponseWriter, r *http.Request) {
	var req models.GameLaunchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.GameID == "" || req.OperatorName == "" {
		http.Error(w, "Game ID and Operator Name are required", http.StatusBadRequest)
		return
	}
	log.Printf("Received Game Launch Request: %+v", req)

	operator, err := postgresClient.FetchOperator(req.OperatorName, req.GameID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid operator / gameID: %v", err), http.StatusBadRequest)
		return
	}
	if !operator.Active {
		http.Error(w, fmt.Sprintf("Inactive game for operator: %s", req.OperatorName), http.StatusBadRequest)
		return
	}
	module, exists := OperatorModules[req.OperatorName]
	if !exists {
		http.Error(w, fmt.Sprintf("Unsupported operator: %s", req.OperatorName), http.StatusBadRequest)
		return
	}

	// Delegate the request to the module
	module.HandleGameLaunch(w, r, req, *operator)
}

// Helper function to generate the game URL
func generateGameURL(baseURL, token, sessionID, currency string) (string, error) {
	// Parse the base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %v", err)
	}

	// Add query parameters
	query := url.Values{}
	query.Add("token", token)
	query.Add("sessionId", sessionID)
	query.Add("currency", currency)
	parsedURL.RawQuery = query.Encode()

	// Return the full URL as a string
	return parsedURL.String(), nil
}

func generatePlayerSession(op models.Operator, token, username, currency string, balance int64) (models.Session, error) {
	session := models.Session{
		ID:           models.GenerateUUID(),
		Token:        token,
		PlayerName:   username,
		Balance:      balance,
		Currency:     currency,
		OperatorName: op.OperatorName,
		CreatedAt:    time.Now(),
	}
	err := redisClient.AddSession(&session)
	return session, err
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/gamelaunch", gameLaunchHandler).Methods("POST")

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
