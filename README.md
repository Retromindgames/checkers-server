# Checkers Game Server

This app is a real-time checkers game platform that uses WebSocket for communication. It consists of several key components:

- **WebSocket API**: Manages player connections and facilitates real-time interactions.
- **Room Worker**: Handles the creation and management of game rooms, to allow players to join and start games.
- **Broadcast Worker**: Sends real-time updates and notifications to players, keeping them informed about game events / info.
- **Game Worker**: Handles the creation and management of games, as well as game related messages.
