package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/Lavizord/checkers-server/config"
	"github.com/Lavizord/checkers-server/logger"
)

var addr = flag.String("addr", ":80", "http service address")

func main() {

	gameEngine := os.Getenv("GAME_ENGINE")
	if gameEngine == "" {
		logger.Default.Fatalf("no GAME_ENGINE env variable defined, exiting")
	}
	urlSufix := os.Getenv("WS_URL_SUFFIX")
	if gameEngine == "" {
		logger.Default.Fatalf("no GAME_ENGINE env variable defined, exiting")
	}
	config.LoadConfig()
	redisConfig := config.Cfg.Redis

	flag.Parse()
	hub := newHub(redisConfig.Addr, redisConfig.User, redisConfig.Password, redisConfig.Tls, gameEngine)
	defer func() {
		hub.Close()
	}()
	go hub.run()

	// we subscribe to our redis broadcast channel.
	hub.SubscribeBroadcast()
	url := fmt.Sprintf("/ws/%s", urlSufix)
	http.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	http.HandleFunc("/ws/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		logger.Default.Fatalf("ListenAndServe error: %v", err)
	}
}
