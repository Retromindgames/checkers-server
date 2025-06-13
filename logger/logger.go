package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Lavizord/checkers-server/redisdb"
)

func NewLogger(redisClient *redisdb.RedisClient) *MultiLogger {
	var redisLogger *RedisLogger
	if redisClient != nil {
		redisLogger = &RedisLogger{RedisClient: redisClient, Key: "logs"}
	}
	return &MultiLogger{
		console: &ConsoleLogger{},
		redis:   redisLogger,
	}
}

type LogTarget int

const (
	LogConsole LogTarget = 1 << iota
	LogRedis
	LogAll = LogConsole | LogRedis
)

type Logger interface {
	Info(format string, v ...any)
	Error(format string, v ...any)
	Fatal(format string, v ...any)
}

type MultiLogger struct {
	console *ConsoleLogger
	redis   *RedisLogger
}

func (m *MultiLogger) SetRedisClient(redisClient *redisdb.RedisClient) {
	if redisClient == nil {
		m.redis = nil
	} else {
		m.redis = &RedisLogger{RedisClient: redisClient, Key: "logs"}
	}
}

// Updated methods to include optional id and identifier params.
func (m *MultiLogger) Info(target LogTarget, id, identifier, format string, v ...any) {
	if target&LogConsole != 0 {
		m.console.Info(id, identifier, format, v...)
	}
	if m.redis != nil && target&LogRedis != 0 {
		m.redis.Info(id, identifier, format, v...)
	}
}

func (m *MultiLogger) Error(target LogTarget, id, identifier, format string, v ...any) {
	if target&LogConsole != 0 {
		m.console.Error(id, identifier, format, v...)
	}
	if m.redis != nil && target&LogRedis != 0 {
		m.redis.Error(id, identifier, format, v...)
	}
}

func (m *MultiLogger) Fatal(target LogTarget, id, identifier, format string, v ...any) {
	if target&LogConsole != 0 {
		m.console.Fatal(id, identifier, format, v...)
	}
	if m.redis != nil && target&LogRedis != 0 {
		m.redis.Fatal(id, identifier, format, v...)
	}
}

type ConsoleLogger struct{}

func (c *ConsoleLogger) Info(id, identifier, f string, v ...any) {
	log.Printf("[INFO] [%s][%s] "+f, append([]any{id, identifier}, v...)...)
}

func (c *ConsoleLogger) Error(id, identifier, f string, v ...any) {
	log.Printf("[ERROR] [%s][%s] "+f, append([]any{id, identifier}, v...)...)
}

func (c *ConsoleLogger) Fatal(id, identifier, f string, v ...any) {
	log.Printf("[FATAL] [%s][%s] "+f, append([]any{id, identifier}, v...)...)
	os.Exit(1)
}

type RedisLogger struct {
	RedisClient *redisdb.RedisClient
	Key         string
}

type RedisLogEntry struct {
	Level      string `json:"level"`
	Message    string `json:"message"`
	ID         string `json:"id,omitempty"`
	Identifier string `json:"identifier,omitempty"`
	Timestamp  string `json:"timestamp"`
}

func (r *RedisLogger) log(level, id, identifier, f string, v ...any) {
	msg := fmt.Sprintf(f, v...)
	entry := RedisLogEntry{
		Level:      level,
		Message:    msg,
		ID:         id,
		Identifier: identifier,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		// fallback: push raw string if marshal fails
		r.RedisClient.Client.RPush(context.Background(), r.Key, fmt.Sprintf("[%s] %s", level, msg))
		return
	}
	r.RedisClient.Client.RPush(context.Background(), r.Key, data)
}

func (r *RedisLogger) Info(id, identifier, f string, v ...any) {
	r.log("INFO", id, identifier, f, v...)
}
func (r *RedisLogger) Error(id, identifier, f string, v ...any) {
	r.log("ERROR", id, identifier, f, v...)
}
func (r *RedisLogger) Fatal(id, identifier, f string, v ...any) {
	r.log("FATAL", id, identifier, f, v...)
	os.Exit(1)
}
