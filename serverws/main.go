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
	hubCheckers := newHub(redisConfig.Addr, redisConfig.User, redisConfig.Password, redisConfig.Tls, "BatalhaDasDamas")
	hubChess := newHub(redisConfig.Addr, redisConfig.User, redisConfig.Password, redisConfig.Tls, "BatalhaDoChess")
	defer func() {
		hubCheckers.Close() // close Redis on exit
		hubChess.Close()
	}()
	go hubCheckers.run()
	go hubChess.run()

	// we subscribe to our redis broadcast channel.
	hubCheckers.SubscribeBroadcast()
	hubChess.SubscribeBroadcast()

	http.HandleFunc("/ws/checkers", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hubCheckers, w, r)
	})
	http.HandleFunc("/ws/chess", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hubChess, w, r)
	})

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		logger.Default.Fatalf("ListenAndServe error: %v", err)
	}
}
