package routes

import (
	"net/http"

	"github.com/Lavizord/checkers-server/api/handlers"
	"github.com/Lavizord/checkers-server/postgrescli"
	"github.com/Lavizord/checkers-server/redisdb"
	"github.com/gorilla/mux"
)

func RegisterRoutes(db *postgrescli.PostgresCli, rd *redisdb.RedisClient) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/api/gamelaunch", handlers.GameLaunchHandler(db, rd)).Methods("POST")
	r.HandleFunc("/api/game/moves", handlers.GameMovesHandler(db, rd)).Methods("POST")

	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/health", healthHandler).Methods("GET")

	return r
}
