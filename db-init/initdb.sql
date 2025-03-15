CREATE TABLE IF NOT EXISTS games (
    id SERIAL PRIMARY KEY,
    operator_name VARCHAR(255) NOT NULL,
    start_date TIMESTAMP NOT NULL DEFAULT NOW(),
    end_date TIMESTAMP,
    moves JSONB NOT NULL DEFAULT '[]',  -- JSON array for moves
    bet_amount DECIMAL(10,2) NOT NULL CHECK (bet_amount >= 0),
    winner VARCHAR(255),  -- Store winner's name or ID
    game_players JSONB NOT NULL DEFAULT '[]'  -- JSON array for players
);
