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
			OperatorName, OperatorGameName, GameName, StartDate, EndDate, Moves, BetAmount, Winner, GamePlayers
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	var gameID int
	err = pc.DB.QueryRow(
		query, 
		game.OperatorIdentifier.OperatorName, 
		game.OperatorIdentifier.OperatorGameName, 
		game.OperatorIdentifier.GameName, 
		game.StartTime, 
		game.EndTime, 
		movesJSON, 
		game.BetValue, 
		game.Winner, 
		playersJSON,
	).Scan(&gameID)

	if err != nil {
		return fmt.Errorf("error inserting game: %w", err)
	}

	fmt.Printf("Game saved with ID: %d\n", gameID)
	return nil
}

// FetchOperator fetches an operator from the database using OperatorName and OperatorGameName
func (pc *PostgresCli) FetchOperator(operatorName, operatorGameName string) (*models.Operator, error) {
	query := `
		SELECT ID, OperatorName, OperatorGameName, GameName, Active, GameBaseUrl, OperatorWalletBaseUrl
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
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("operator not found with OperatorName=%s and OperatorGameName=%s", operatorName, operatorGameName)
		}
		return nil, fmt.Errorf("error fetching operator: %w", err)
	}

	return &operator, nil
}
