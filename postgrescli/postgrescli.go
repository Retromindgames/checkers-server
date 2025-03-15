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

/*  USAGE EXAMPLE

    // Create a new PostgresCli instance
	pgCli, err := NewPostgresCli("sa", "checkersdb", "checkers", "localhost", "5432")
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}
	defer pgCli.Close()

	// Example game data
	game := Game{
		ID:              "123",
		OperatorName:    "Operator A",
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(time.Hour * 1), // just for example
		BetValue:        50.00,
		Winner:          "Player 1",
		TimerSetting:    "60",
		Players: []GamePlayer{
			{ID: "player1", Token: "token1", Name: "Player 1", Timer: 60, Color: "red", SessionID: "session1", NumPieces: 12},
			{ID: "player2", Token: "token2", Name: "Player 2", Timer: 60, Color: "black", SessionID: "session2", NumPieces: 12},
		},
		Moves: []Move{
			{PlayerID: "player1", PieceID: "piece1", From: "A1", To: "A2", IsCapture: false, IsKinged: false},
			{PlayerID: "player2", PieceID: "piece2", From: "B1", To: "B2", IsCapture: true, IsKinged: false},
		},
	}

	// Save the game to the database
	if err := pgCli.SaveGame(game); err != nil {
		log.Fatal("Error saving game: ", err)
	}
*/

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
