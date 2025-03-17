package postgrescli

import (
	"checkers-server/models"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq"
)

type PostgresCli struct {
	DB *sql.DB
}

func NewPostgresCli(user, password, dbname, host, port string) (*PostgresCli, error) {
	connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable", user, password, dbname, host, port)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Ping to make sure the connection is valid
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return &PostgresCli{DB: db}, nil
}

// Close method to close the database connection
func (pc *PostgresCli) Close() {
	pc.DB.Close()
}

// SaveGame method to save a game to the database
func (pc *PostgresCli) SaveGame(game models.Game) error {
	// Convert moves to JSONB
	movesJSON, err := json.Marshal(game.Moves)
	if err != nil {
		return fmt.Errorf("error marshalling moves: %w", err)
	}

	// Convert game_players to JSONB
	playersJSON, err := json.Marshal(game.Players)
	if err != nil {
		return fmt.Errorf("error marshalling players: %w", err)
	}

	// SQL query to insert the game data
	query := `
		INSERT INTO games (
			operator_name, start_date, end_date, moves, bet_amount, winner, game_players
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	var gameID int
	err = pc.DB.QueryRow(query, game.OperatorName, game.StartTime, game.EndTime, movesJSON, game.BetValue, game.Winner, playersJSON).Scan(&gameID)
	if err != nil {
		return fmt.Errorf("error inserting game: %w", err)
	}

	fmt.Printf("Game saved with ID: %d\n", gameID)
	return nil
}

// FetchOperator fetches an operator from the database using operator_name and game_name
func (pc *PostgresCli) FetchOperator(operatorName, operatorGameName string) (*models.Operator, error) {
	query := `
		SELECT id, operator_name, operator_game_name, game_name, active, game_base_url, operator_wallet_base_url
		FROM operators
		WHERE operator_name = $1 AND operator_game_name = $2
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
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("operator not found with operator_name=%s and game_name=%s", operatorName, operatorGameName)
		}
		return nil, fmt.Errorf("error fetching operator: %w", err)
	}

	return &operator, nil
}
