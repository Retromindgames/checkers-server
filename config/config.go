package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

/*
	{
	"redis": {
		"addr": "redis:6379",
		"db": 0
	},
	"services": {
		"wsapi": { "ports": [8080, 8081, 8082] },
		"pstatusworker": {},
		"roomworker": {},
		"gameworker": {
			"timer": 15,
			"timer_settings": "reset", 	// Options: "reset" or "cumulative"
			"pieces_in_match": 10 				// Number of pieces in the match
		}
	}
	}
*/

type Config struct {
	Redis struct {
		Addr string `json:"addr"`
		DB   int    `json:"db"`
	} `json:"redis"`
	Services map[string]struct {
		Ports          []int  `json:"ports,omitempty"`
		Timer          int    `json:"timer,omitempty"`
		TimerSetting   string `json:"timer_setting,omitempty"` // New field
		PiecesInMatch  int    `json:"pieces_in_match,omitempty"` // New field
	} `json:"services"`
}
// Global config instance
var Cfg Config

func LoadConfig() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("[config.go] - CONFIG_PATH not set")
	} 
	fmt.Println("[config.go] - Config path is:", configPath)
	
	file, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("[config.go] -Error opening config file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&Cfg); err != nil {
		log.Fatalf("[config.go] - Error decoding JSON: %v", err)
	}
	log.Printf("[config.go] -Config loaded: %+v", Cfg)
}