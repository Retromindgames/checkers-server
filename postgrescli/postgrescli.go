package postgrescli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Lavizord/checkers-server/models"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

type PostgresCli struct {
	DB *sql.DB
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

	return &PostgresCli{DB: db}, nil
}

func (pc *PostgresCli) Close() {
	pc.DB.Close()
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
		session.OperatorIdentifier.OperatorName,
		session.OperatorIdentifier.OperatorGameName,
		session.OperatorIdentifier.GameName,
	)
	//duration := time.Since(start)
	//log.Printf("[Postgres metric] - SaveSession Insert took %v", duration)

	if err != nil {
		log.Printf("[PostgresCli] - error saving session: %v", err)
		return fmt.Errorf("exec session insert: %w", err)
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
		game.OperatorIdentifier.OperatorName,
		game.OperatorIdentifier.OperatorGameName,
		game.OperatorIdentifier.GameName,
		game.StartTime,
		game.EndTime,
		string(movesJSON),
		game.BetValue,
		game.Winner,
		string(playersJSON),
		game.OperatorIdentifier.WinFactor,
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

// FetchOperator fetches an operator from the database using OperatorName and OperatorGameName
func (pc *PostgresCli) FetchOperator(operatorName, operatorGameName string) (*models.Operator, error) {
	query := `
		SELECT ID, OperatorName, OperatorGameName, GameName, Active, GameBaseUrl, OperatorWalletBaseUrl, WinFactor
		FROM operators
		WHERE OperatorName = $1 AND OperatorGameName = $2
	`
	row := pc.DB.QueryRow(query, operatorName, operatorGameName)

	var operator models.Operator
	err := row.Scan(
		&operator.ID,
		&operator.OperatorName,
		&operator.OperatorGameName,
		&operator.GameName,
		&operator.Active,
		&operator.GameBaseUrl,
		&operator.OperatorWalletBaseUrl,
		&operator.WinFactor,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("operator not found with OperatorName=%s and OperatorGameName=%s", operatorName, operatorGameName)
		}
		return nil, fmt.Errorf("error fetching operator: %w", err)
	}

	return &operator, nil
}

func (pc *PostgresCli) FetchAllOperators() ([]models.Operator, error) {
	query := `
		SELECT ID, OperatorName, OperatorGameName, GameName, Active, GameBaseUrl, OperatorWalletBaseUrl, WinFactor
		FROM operators
		WHERE active = true
	`

	rows, err := pc.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying operators: %w", err)
	}
	defer rows.Close()

	var operators []models.Operator
	for rows.Next() {
		var operator models.Operator
		err := rows.Scan(
			&operator.ID,
			&operator.OperatorName,
			&operator.OperatorGameName,
			&operator.GameName,
			&operator.Active,
			&operator.GameBaseUrl,
			&operator.OperatorWalletBaseUrl,
			&operator.WinFactor,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning operator: %w", err)
		}
		operators = append(operators, operator)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return operators, nil
}
