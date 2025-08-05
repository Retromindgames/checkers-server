package postgrescli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/postgrescli/ent"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

type PostgresCli struct {
	DB     *sql.DB
	EntCli *ent.Client
}

func NewPostgresCli(user, password, dbname, host, port string, ssl bool) (*PostgresCli, error) {
	connStr := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s binary_parameters=yes",
		user, password, dbname, host, port)

	if ssl {
		connStr += " sslmode=require"
	} else {
		connStr += " sslmode=disable"
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetConnMaxLifetime(time.Minute * 5)
	db.SetMaxOpenConns(60)
	db.SetMaxIdleConns(20)

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Ent client
	entClient, err := ent.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open ent client: %w", err)
	}

	return &PostgresCli{
		DB:     db,
		EntCli: entClient,
	}, nil
}

func (pc *PostgresCli) Close() {
	pc.DB.Close()
	pc.EntCli.Close()
}

func (pc *PostgresCli) CreateDb() error {
	//TODO: Auto-create schema remove in production, replace with migrations
	if err := pc.EntCli.Schema.Create(context.Background()); err != nil {
		return fmt.Errorf("failed to run ent schema migration: %w", err)
	}
	return nil
}

func (pc *PostgresCli) SaveSession(session models.Session) error {
	query := `
		INSERT INTO sessions (
			SessionId, Token, PlayerName, Currency, OperatorBaseUrl, CreatedAt, OperatorName, OperatorGameName, GameName
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	//start := time.Now()

	stmt, err := pc.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("prepare session insert: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		session.ID,
		session.Token,
		session.PlayerName,
		session.Currency,
		session.OperatorBaseUrl,
		session.CreatedAt,
	)
	//duration := time.Since(start)
	//log.Printf("[Postgres metric] - SaveSession Insert took %v", duration)

	if err != nil {
		log.Printf("[PostgresCli] - error saving session: %v", err)
		return fmt.Errorf("exec session insert: %w", err)
	}
	return nil
}

func (pc *PostgresCli) SaveSessionNew(session models.Session) error {
	ctx := context.Background()
	_, err := pc.EntCli.Session.
		Create().
		SetToken(session.Token).
		SetClientID(session.ID).
		SetDemo(false).
		Save(ctx)

	if err != nil {
		log.Printf("[PostgresCli] - error saving session: %v", err)
		return fmt.Errorf("ent: save session: %w", err)
	}
	return nil
}

func (pc *PostgresCli) SaveTransaction(transaction models.Transaction) error {
	query := `
		INSERT INTO transactions (
			TransactionID, SessionID, Type, Amount, Currency, Platform, Operator, Client, Game, Status, Description, RoundID, Timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	stmt, err := pc.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("prepare transaction insert: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		transaction.ID,
		transaction.SessionID,
		transaction.Type,
		transaction.Amount,
		transaction.Currency,
		transaction.Platform,
		transaction.Operator,
		transaction.Client,
		transaction.Game,
		transaction.Status,
		transaction.Description,
		transaction.RoundID,
		transaction.Timestamp,
	)
	if err != nil {
		log.Printf("[PostgresCli] - error saving transaction: %v", err)
		return fmt.Errorf("exec transaction insert: %w", err)
	}
	return nil
}

func (pc *PostgresCli) SaveGame(game models.Game, reason string) error {
	movesJSON, err := json.Marshal(game.Moves)
	if err != nil {
		log.Printf("[PostgresCli] - error marsheling moves to save game: %v", err)
		return fmt.Errorf("marshal moves: %w", err)
	}
	playersJSON, err := json.Marshal(game.Players)
	if err != nil {
		log.Printf("[PostgresCli] - error marsheling players to save game: %v", err)
		return fmt.Errorf("marshal players: %w", err)
	}

	query := `
		INSERT INTO games (
			ID, OperatorName, OperatorGameName, GameName, StartDate, EndDate, Moves, BetAmount, Winner, GamePlayers, WinFactor, NumMoves, GameOverReason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	stmt, err := pc.DB.Prepare(query)
	if err != nil {
		log.Printf("[PostgresCli] - error preparing game insert: %v", err)
		return fmt.Errorf("prepare game insert: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		game.ID,
		//game.OperatorIdentifier.OperatorName,
		//game.OperatorIdentifier.OperatorGameName,
		//game.OperatorIdentifier.GameName,
		game.StartTime,
		game.EndTime,
		string(movesJSON),
		game.BetValue,
		game.Winner,
		string(playersJSON),
		//game.OperatorIdentifier.WinFactor,
		len(game.Moves),
		reason,
	)
	if err != nil {
		log.Printf("[PostgresCli] - error saving game: %v", err)
		return fmt.Errorf("exec game insert: %w", err)
	}
	return nil
}

func (pc *PostgresCli) FetchGameMoves(gameID string) ([]models.Move, error) {
	query := `SELECT moves FROM games WHERE id = $1`
	log.Printf("Fetching moves for gameID: %s", gameID) // not [%s]
	parsedUUID, err := uuid.Parse(gameID)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %w", err)
	}
	log.Printf("Fetching moves for gameID: %s", gameID) // not [%s]

	var movesJSON []byte
	err = pc.DB.QueryRow(query, parsedUUID).Scan(&movesJSON)
	if err != nil {
		return nil, fmt.Errorf("error fetching moves for game %s: %w", gameID, err)
	}

	var moves []models.Move
	if err := json.Unmarshal(movesJSON, &moves); err != nil {
		return nil, fmt.Errorf("error unmarshalling moves JSON: %w", err)
	}

	return moves, nil
}
