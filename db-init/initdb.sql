DO $$
BEGIN
    -- Check if the table exists
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_name = 'operators'
    ) THEN
        -- Create the table if it doesn't exist
        CREATE TABLE operators (
            id SERIAL PRIMARY KEY,
            operator_name VARCHAR(255) NOT NULL,         -- Name of the operator
            operator_game_name VARCHAR(255) NOT NULL,    -- Name of the game for the operator
            game_name VARCHAR(255) NOT NULL,             -- Internal game name.
            active BOOLEAN NOT NULL DEFAULT TRUE,        -- Whether the operator is active or not, default to TRUE (1)
            game_base_url VARCHAR(255) NOT NULL,                       -- gamelaunch base url
            operator_wallet_base_url VARCHAR(255) NOT NULL
        );

        -- Insert a row into the table after creating it
        INSERT INTO operators (operator_name, operator_game_name, game_name, game_base_url, operator_wallet_base_url)
        VALUES (
            'SokkerDuel',
            'damasSokkerDuel',
            'BatalhaDasDamas',
            'https://miguelclg.github.io/Damas/docs',
            'http://88.99.49.131:3000'
        );
    END IF;
END $$;

/*
 Games table, to store the games when they are finished.
 Operator name identifies the casino that is operating the game.
*/
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