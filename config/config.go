package config

import (
	"encoding/json"
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
			"timer_settings": "reset", 			// Options: "reset" or "cumulative"
			"pieces_in_match": 10 				// Number of pieces in the match
		}
	}
	}
*/

type Config struct {
	Redis struct {
		Addr     string `json:"addr"`
		DB       int    `json:"db"`
		User     string `json:"user,omitempty"`
		Password string `json:"password,omitempty"`
	} `json:"redis"`
	Postgres struct {
		User     string `json:"user"`
		Password string `json:"password"`
		DBName   string `json:"dbname"`
		Host     string `json:"host"`
		Port     string `json:"port"`
	} `json:"postgres"`
	Services map[string]struct {
		Ports         []int  `json:"ports,omitempty"`
		Timer         int    `json:"timer,omitempty"`
		TimerSetting  string `json:"timer_setting,omitempty"`
		PiecesInMatch int    `json:"pieces_in_match,omitempty"`
	} `json:"services"`
}

// Global config instance
var Cfg Config

func LoadConfig() {
	if os.Getenv("PROD") != "" {
		log.Println("[config.go] - Loading config from file and environment variables (production mode)")
		loadConfigFromFile() // we load it from file.
		loadConfigFromEnv()  // then we override with vens, to make the transition smoother
	} else {
		log.Println("[config.go] - Loading config from file (development mode)")
		loadConfigFromFile()
	}
}

func loadConfigFromFile() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("[config.go] - CONFIG_PATH not set")
	}

	file, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("[config.go] - Error opening config file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&Cfg); err != nil {
		log.Fatalf("[config.go] - Error decoding JSON: %v", err)
	}
	log.Println("[config.go] - Config file successfully loaded and decoded.")
}

func loadConfigFromEnv() {
	Cfg.Redis.Addr = os.Getenv("REDIS_ADDR")
	Cfg.Redis.DB = 0
	Cfg.Redis.User = os.Getenv("REDIS_USER")
	Cfg.Redis.Password = os.Getenv("REDIS_PASS")
	log.Println("[config.go] - Loaded Redis config from environment variables:")
	log.Printf("  Redis Address: %s\n", Cfg.Redis.Addr)
	log.Printf("  Redis DB: %d\n", Cfg.Redis.DB)
	log.Printf("  Redis User: %s\n", Cfg.Redis.User)
	log.Printf("  Redis Password: %s\n", Cfg.Redis.Password)
}
