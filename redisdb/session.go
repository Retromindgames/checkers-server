package redisdb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Lavizord/checkers-server/models"
)

func (r *RedisClient) AddSession(session *models.Session) error {
	ctx := context.Background()
	ttl := 6 * time.Hour

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("[RedisClient] (Session) - failed to serialize session: %v", err)
	}

	operatorIdentifierData, err := json.Marshal(session.OperatorIdentifier)
	if err != nil {
		return fmt.Errorf("[RedisClient] (Session) - failed to serialize operator identifier: %v", err)
	}

	// Use consistent hash tag for all keys to avoid CROSSSLOT errors
	hashTag := fmt.Sprintf("{%s}", session.ID) // or use operator for operator index

	sessionKey := fmt.Sprintf("session:%s", hashTag)
	tokenKey := fmt.Sprintf("session_token:%s:%s", hashTag, session.Token)
	indexKey := fmt.Sprintf("session_index:%s:%s:%s:%s",
		hashTag, // hash tag for index key
		session.OperatorIdentifier.OperatorName,
		session.PlayerName,
		session.Currency,
	)

	pipe := r.Client.TxPipeline()

	pipe.HSet(ctx, sessionKey, map[string]interface{}{
		"id":                  session.ID,
		"token":               session.Token,
		"player_name":         session.PlayerName,
		"currency":            session.Currency,
		"operator_identifier": string(operatorIdentifierData),
		"operator_base_url":   session.OperatorBaseUrl,
		"created_at":          session.CreatedAt.Format(time.RFC3339),
		"data":                string(data),
	})
	pipe.Expire(ctx, sessionKey, ttl)
	pipe.Set(ctx, tokenKey, session.ID, ttl)
	pipe.Set(ctx, indexKey, session.ID, ttl)
	//fmt.Printf("[RedisClient] Keys used in pipeline:\n")
	//fmt.Printf("  sessionKey: %s\n", sessionKey)
	//fmt.Printf("  tokenKey:   %s\n", tokenKey)
	//fmt.Printf("  indexKey:   %s\n", indexKey)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("[RedisClient] (Session) - failed to store session: %v", err)
	}
	return nil
}

func (r *RedisClient) RemoveSession(sessionID string) error {
	ctx := context.Background()
	hashTag := fmt.Sprintf("{%s}", sessionID)
	sessionKey := fmt.Sprintf("session:%s", hashTag)

	fields, err := r.Client.HMGet(ctx, sessionKey,
		"token",
		"operator_identifier",
		"player_name",
		"currency",
	).Result()
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to retrieve session metadata: %v", err)
	}
	if fields[0] == nil || fields[1] == nil || fields[2] == nil || fields[3] == nil {
		return fmt.Errorf("[RedisClient] - incomplete session metadata")
	}

	token := fields[0].(string)
	opIdJSON := fields[1].(string)
	playerName := fields[2].(string)
	currency := fields[3].(string)

	var opIdentifier models.OperatorIdentifier
	if err := json.Unmarshal([]byte(opIdJSON), &opIdentifier); err != nil {
		return fmt.Errorf("[RedisClient] - failed to unmarshal operator identifier: %v", err)
	}

	tokenKey := fmt.Sprintf("session_token:%s:%s", hashTag, token)
	indexKey := fmt.Sprintf("session_index:%s:%s:%s:%s",
		hashTag,
		opIdentifier.OperatorName,
		playerName,
		currency,
	)

	pipe := r.Client.TxPipeline()
	pipe.Del(ctx, sessionKey)
	pipe.Del(ctx, tokenKey)
	pipe.Del(ctx, indexKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to delete session data: %v", err)
	}
	return nil
}

func (r *RedisClient) GetSessionByID(sessionID string) (*models.Session, error) {
	ctx := context.Background()
	sessionKey := fmt.Sprintf("session:{%s}", sessionID)

	fields, err := r.Client.HMGet(ctx, sessionKey, "data").Result()
	if err != nil || fields[0] == nil {
		return nil, fmt.Errorf("[RedisClient] - session data not found for %s", sessionID)
	}

	var session models.Session
	data, ok := fields[0].(string)
	if !ok {
		return nil, fmt.Errorf("[RedisClient] - session data is not a valid string")
	}

	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("[RedisClient] - failed to unmarshal session data: %v", err)
	}

	return &session, nil
}

func (r *RedisClient) GetSessionByToken(token string) (*models.Session, error) {
	ctx := context.Background()

	// Search keys matching pattern if token hash tag isn't known yet
	pattern := fmt.Sprintf("session_token:{*}:%s", token)
	keys, err := r.Client.Keys(ctx, pattern).Result()
	if err != nil || len(keys) == 0 {
		return nil, fmt.Errorf("[RedisClient] - token key not found: %s", token)
	}

	// Extract sessionID from key (between `{}` in hash tag)
	var sessionID string
	_, err = fmt.Sscanf(keys[0], "session_token:{%s}:"+token, &sessionID)
	if err != nil {
		return nil, fmt.Errorf("[RedisClient] - failed to extract session ID from token key: %v", err)
	}
	sessionID = strings.TrimRight(sessionID, "}") // remove trailing }
	return r.GetSessionByID(sessionID)
}

func (r *RedisClient) GetSessionByOperatorPlayerCurrency(operator, playerName, currency string) (*models.Session, error) {
	ctx := context.Background()
	pattern := fmt.Sprintf("session_index:*:%s:%s:%s", operator, playerName, currency)

	var cursor uint64
	var sessionKey string
	for {
		keys, nextCursor, err := r.Client.Scan(ctx, cursor, pattern, 10).Result()
		if err != nil {
			return nil, fmt.Errorf("[RedisClient] - scan error: %v", err)
		}
		if len(keys) > 0 {
			sessionKey = keys[0]
			break
		}
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}

	if sessionKey == "" {
		return nil, fmt.Errorf("[RedisClient] - session not found for operator: %s, player: %s, currency: %s", operator, playerName, currency)
	}

	// Extract session ID from key: session_index:{sessionID}:...
	start := strings.Index(sessionKey, "{")
	end := strings.Index(sessionKey, "}")
	if start == -1 || end == -1 || end <= start+1 {
		return nil, fmt.Errorf("[RedisClient] - failed to extract session ID from key: %s", sessionKey)
	}
	sessionID := sessionKey[start+1 : end]

	return r.GetSessionByID(sessionID)
}

func (r *RedisClient) RefreshSessionTTL(session *models.Session, ttl time.Duration) error {
	ctx := context.Background()

	hashTag := fmt.Sprintf("{%s}", session.ID)
	sessionKey := fmt.Sprintf("session:%s", hashTag)
	tokenKey := fmt.Sprintf("session_token:%s:%s", hashTag, session.Token)
	indexKey := fmt.Sprintf("session_index:%s:%s:%s:%s",
		hashTag,
		session.OperatorIdentifier.OperatorName,
		session.PlayerName,
		session.Currency,
	)

	pipe := r.Client.TxPipeline()
	pipe.Expire(ctx, sessionKey, ttl)
	pipe.Expire(ctx, tokenKey, ttl)
	pipe.Expire(ctx, indexKey, ttl)

	_, err := pipe.Exec(ctx)
	return err
}
