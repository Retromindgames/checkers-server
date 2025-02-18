# Checkers Game Server

This app is a real-time checkers game platform that uses WebSocket for communication. It consists of several key components:

- **WebSocket API**: Manages player connections and facilitates real-time interactions.
- **Player Status Worker**: Monitors and manages player statuses.
- **Room Worker**: Handles the creation and management of game rooms, to allow players to join and start games.
- **Broadcast Worker**: Sends real-time updates and notifications to players, keeping them informed about game events / info.

# Run with Docker Compose
## Prerequisites
- Docker
- Docker Compose

## Docker Compose Commands

1. **Build and start the app**:
   ```bash
      docker-compose up --build
   ```
## WebSocket Connection

The application supports WebSocket connections, allowing real-time communication. You can connect to the WebSocket server using the following URL:

ws://localhost:80080

## Redis Cliente

To access the Redis client inside a Docker container running Redis 9, use the following command:

   ```bash
      docker exec -it <container_name_or_id> redis-cli
   ```

Replace <container_name_or_id> with the actual container name or ID of the Redis container. 
This will open the Redis CLI within the container, allowing you to interact with the Redis server.

### Usefull redis commands

Here are two useful Redis commands:

KEYS – Retrieves all keys matching a pattern:

   ```bash
      KEYS *
   ```

This will return all the keys in the Redis database. You can also use a pattern, like KEYS player:*, to search for keys that match a specific pattern.

HGET – Retrieves the value of a specific field in a Redis hash:

   ```bash
   HGET players <player_id>
   ```

Replace <player_id> with the actual ID of the player. This will return the value associated with that specific field within the players hash.



