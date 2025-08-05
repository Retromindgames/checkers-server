package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Lavizord/checkers-server/config"
	"github.com/Lavizord/checkers-server/interfaces"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/postgrescli/ent"
	"github.com/Lavizord/checkers-server/redisdb"

	"github.com/gorilla/mux"
)

var postgresClient *postgrescli.PostgresCli
var redisClient *redisdb.RedisClient
var name = "restapi"

func init() {
	config.LoadConfig()

	redisConData := config.Cfg.Redis
	client, err := redisdb.NewRedisClient(redisConData.Addr, redisConData.User, redisConData.Password, redisConData.Tls)
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
		config.Cfg.Postgres.Ssl,
	)
	if err != nil {
		logger.Default.Fatalf("[PostgreSQL] Error initializing POSTGRES client: %v\n", err)
	}
	postgresClient = sqlcliente

	err = postgresClient.CreateDb()
	if err != nil {
		logger.Default.Fatalf("error creating db: %v", err)
	} else {
		logger.Default.Infof("created db...")
	}
	err = postgresClient.SeedDb()
	if err != nil {
		logger.Default.Fatalf("error seeding db: %v", err)
	} else {
		logger.Default.Infof("seeded db...")
	}

	_, err = CacheOperators()
	if err != nil {
		logger.Default.Fatalf("error caching operators in redis: %v", err)
	} else {
		logger.Default.Infof("cached operators in redis...")
	}

	_, err = CacheGameConfigs()
	if err != nil {
		logger.Default.Fatalf("error caching game configs in redis: %v", err)
	} else {
		logger.Default.Infof("cached game configs in redis...")
	}

	logger.Default.Info("initialized api...")
}

func CacheOperators() ([]*ent.Operator, error) {
	op, err := postgresClient.GetOperators()
	if err != nil {
		return nil, err
	}
	if len(op) == 0 || op == nil {
		return nil, fmt.Errorf("failed to cache operators, none found in database.")
	}

	err = redisClient.AddOperators(op)
	if err != nil {
		return nil, fmt.Errorf("failed adding operators in redis.")
	}

	return op, nil
}

func CacheGameConfigs() ([]*ent.GameConfig, error) {
	gc, err := postgresClient.GetAllGameConfigs()
	if err != nil {
		return nil, err
	}
	if len(gc) == 0 || gc == nil {
		return nil, fmt.Errorf("failed to cache game configs, none found in database.")
	}

	err = redisClient.AddGameConfigs(gc)
	if err != nil {
		return nil, fmt.Errorf("failed adding game configs in redis.")
	}

	return gc, nil
}

func gameLaunchHandler(w http.ResponseWriter, r *http.Request) {
	var req models.GameLaunchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithJSON(w, http.StatusBadRequest, models.GameLaunchResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}
	if req.GameID == "" || req.OperatorName == "" || req.Currency == "" {
		respondWithJSON(w, http.StatusBadRequest, models.GameLaunchResponse{
			Success: false,
			Message: "Game ID, Operator Name and Currency are required",
		})
		return
	}

	config, err := redisClient.GetGameConfig(req.GameID, req.OperatorName, req.Currency)
	if err != nil {
		config, err := postgresClient.GetGameConfig(req.GameID, req.OperatorName, req.Currency)
		if err != nil {
			logger.Default.Warnf("[GameLaunchHandler] - error fetching the gameconfig from sql: %v", err)
			respondWithJSON(w, http.StatusBadRequest, models.GameLaunchResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid operator / gameID: %v", err),
			})
			return
		}
		redisClient.AddGameConfig(config)
	}

	module, exists := interfaces.PlatformModules[config.PlatformName]
	if !exists {
		respondWithJSON(w, http.StatusBadRequest, models.GameLaunchResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported platform: %s", req.OperatorName),
		})
		logger.Default.Warnf("[GameLaunchHandler] - Unsupported platform: %s", config.PlatformName)
		return
	}

	// Delegate the request to the module
	module.HandleGameLaunch(w, r, req, *operator, redisClient, postgresClient)
}

func gameMovesHandler(w http.ResponseWriter, r *http.Request) {
	var req models.GameMovesRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GameID == "" {
		respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Invalid or missing game_id",
		})
		return
	}

	log.Printf("Fetching moves for gameID: [%s]", req.GameID)
	moves, err := postgresClient.FetchGameMoves(req.GameID)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Failed to fetch moves :" + err.Error(),
		})
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"moves":   moves,
	})
}

// Utility function to respond with JSON
func respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func registerRoutes(r *mux.Router) {
	r.HandleFunc("/api/gamelaunch", gameLaunchHandler).Methods("POST")
	r.HandleFunc("/api/game/moves", gameMovesHandler).Methods("POST")

	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/health", healthHandler).Methods("GET")
}

func main() {
	defer func() {
		if redisClient != nil {
			redisClient.CloseRedisClient()
		}
		if postgresClient != nil {
			postgresClient.Close()
		}
	}()

	router := mux.NewRouter()
	registerRoutes(router)

	port := config.FirstPortFromConfig(name)
	addrs := fmt.Sprintf(":%d", port)

	log.Printf("[API] - HTTP server starting on %d...", port)
	log.Fatal(http.ListenAndServe(addrs, router))

}
