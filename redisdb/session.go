package redisdb

import (
	"checkers-server/models"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (r *RedisClient) AddSession(session *models.Session) error {
	ctx := context.Background()

	// Ensure CreatedAt is set if not already
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

	// Store session data in a hash
	sessionKey := fmt.Sprintf("session:%s", session.ID)
	err = r.Client.HSet(ctx, sessionKey, map[string]interface{}{
		"id":                  session.ID,
		"token":               session.Token,
		"player_name":         session.PlayerName,
		"balance":             session.Balance,
		"currency":            session.Currency,
		"operator_identifier": string(operatorIdentifierData),
		"operator_base_url":   session.OperatorBaseUrl,
		"created_at":          session.CreatedAt.Format(time.RFC3339), // Store timestamp as string
		"data":                string(data),                           // Store full session JSON
	}).Err()
	if err != nil {
		return fmt.Errorf("[RedisClient] (Session) - failed to store session data: %v", err)
	}

	// Store session index for fast lookup
	indexKey := fmt.Sprintf("session_index:%s:%s:%s", session.OperatorBaseUrl, session.PlayerName, session.Currency)
	err = r.Client.Set(ctx, indexKey, session.ID, 0).Err() // No expiration
	if err != nil {
		return fmt.Errorf("[RedisClient] (Session) - failed to store session index: %v", err)
	}

	return nil
}

func (r *RedisClient) RemoveSession(sessionID string) error {
	ctx := context.Background()
	sessionKey := fmt.Sprintf("session:%s", sessionID)
	// Retrieve operator, player, and currency to delete the index
	fields, err := r.Client.HMGet(ctx, sessionKey, "operator_base_url", "player_name", "currency").Result()
	if err != nil || fields[0] == nil || fields[1] == nil || fields[2] == nil {
		return fmt.Errorf("[RedisClient] - failed to retrieve session metadata for deletion: %v", err)
	}
	indexKey := fmt.Sprintf("session_index:%s:%s:%s", fields[0], fields[1], fields[2])

	// Delete both session and index
	pipe := r.Client.TxPipeline()
	pipe.Del(ctx, sessionKey)
	pipe.Del(ctx, indexKey)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("[RedisClient] - failed to delete session and index: %v", err)
	}
	return nil
}

func (r *RedisClient) GetSessionByID(sessionID string) (*models.Session, error) {
    ctx := context.Background()
    sessionKey := fmt.Sprintf("session:%s", sessionID)

    // Retrieve session fields (including 'data' and any other necessary fields)
    fields, err := r.Client.HMGet(ctx, sessionKey, "data").Result()
    if err != nil {
        return nil, fmt.Errorf("[RedisClient] - failed to retrieve session %s: %v", sessionID, err)
    }
    if fields[0] == nil {
        return nil, fmt.Errorf("[RedisClient] - session data not found for %s", sessionID)
    }

    // Unmarshal 'data' field into the session object
    var session models.Session
    data, ok := fields[0].(string) // Ensure correct type assertion
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
	tokenKey := fmt.Sprintf("session_token:%s", token)

	// Get session ID directly using the index
	sessionID, err := r.Client.Get(ctx, tokenKey).Result()
	if err != nil {
		return nil, fmt.Errorf("[RedisClient] - session not found for token: %s", token)
	}
	// Retrieve session data
	return r.GetSessionByID(sessionID)
}


func (r *RedisClient) GetSessionByOperatorPlayerCurrency(operator, playerName, currency string) (*models.Session, error) {
	ctx := context.Background()
	indexKey := fmt.Sprintf("session_index:%s:%s:%s", operator, playerName, currency)

	// Fetch session ID using index
	sessionID, err := r.Client.Get(ctx, indexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("[RedisClient] - session not found for operator: %s, player: %s, currency: %s", operator, playerName, currency)
	}

	// Fetch full session data
	return r.GetSessionByID(sessionID)
}
