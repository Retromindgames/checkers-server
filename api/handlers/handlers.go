package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Lavizord/checkers-server/interfaces"
	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
)

func GameLaunchHandler(postgresClient *postgrescli.PostgresCli, redisClient *redisdb.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		module.HandleGameLaunch(w, r, req, config, redisClient, postgresClient)
	}
}

func GameMovesHandler(postgresClient *postgrescli.PostgresCli, redisClient *redisdb.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
}
