package main

import (
	"flag"
	"net/http"

	"github.com/Lavizord/checkers-server/config"
	"github.com/Lavizord/checkers-server/logger"
)

var addr = flag.String("addr", ":80", "http service address")

func main() {

	config.LoadConfig()
	redisConfig := config.Cfg.Redis

	http.HandleFunc("/ws/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	flag.Parse()
	hub := newHub(redisConfig.Addr, redisConfig.User, redisConfig.Password, redisConfig.Tls)
	defer hub.Close() // close Redis on exit

	go hub.run()
	// we subscribe to our redis broadcast channel.
	hub.SubscribeBroadcast()

	http.HandleFunc("/ws/checkers", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		logger.Default.Fatalf("ListenAndServe error: %v", err)
	}
}
