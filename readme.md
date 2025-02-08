# Checkers Game Server

This is a checkers game server in Go, supporting WebSocket connections for real-time interactions between players.

## Requirements

To run the server, make sure you have the following installed:

- [Go](https://golang.org/dl/) (version 1.23.6 or higher)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)

## Setup

1. Clone the repository:

   ```bash
   git clone https://github.com/Lavizord/go-websocket-checkers
   cd checkers-server
   ```

2. Install necessary dependencies:

   ```bash
   go mod download
   go mod tidy
   ```

## Running the Server

    ```bash
    go run .
    ```

    The server will start listening on port 8080 (default).
    You can change the port by modifying the code in `main.go`.

## WebSocket API

The server uses WebSockets to handle client connections and messages. The WebSocket connection URL is:

```
ws://localhost:8080
```

Clients (players) can send messages like joining the queue, making moves, and more.

### Available Commands

- **`join_queue`**: Join the waiting queue for a game.
- **`leave_queue`**: Leave the waiting queue.
- **`move_piece`**: Move a piece on the board.
- **`send_message`**: Send a chat message to the opponent.

For an updated list check the messages-examples.md file.

## Testing the Server

You can use tools like [Postman](https://www.postman.com/) to test WebSocket connections.

Example message to join the queue:

```json
{
  "command": "join_queue",
  "value": 10
}
```
