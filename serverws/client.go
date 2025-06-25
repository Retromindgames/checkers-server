package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Lavizord/checkers-server/logger"
	"github.com/Lavizord/checkers-server/messages"
	"github.com/Lavizord/checkers-server/models"
	"github.com/Lavizord/checkers-server/redisdb"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 2 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var prodVenv string

func init() {
	prodVenv = os.Getenv("PROD")
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		//log.Println("WebSocket request Origin:", origin)
		return true
		if prodVenv == "" {
			return true
		}
		return origin == "http://localhost:8060" || origin == "https://play.retromindgames.pt/"
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// This represents our player data.
	player *models.Player

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Context, to be used to close
	ctx    context.Context
	cancel context.CancelFunc
}

// CloseConnection cancels the client context
//
// This will trigger the defer in readPump and / or writePump, to handle the
// websocket disconnect.
func (c *Client) CloseConnection() {
	c.cancel() // triggers context cancellation
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		logger.Default.Infof("[Client] - readPump defer for session: %v", c.player.ID)
		c.cancel()
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			logger.Default.Warnf("[Client] - readPump conn ReadMessage failed for session: %v with err: %v", c.player.ID, err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Default.Errorf("[Client] - readPump socket with unexpected close for session: %v with err: %v", c.player.ID, err)
			}
			break
		}
		// I question if this message parsed is really needed. Right now it only helps to make sure some of the values received
		// are the right ones or not, and transforms or message into the right type.
		parsedmessage, err := messages.ParseMessage(message)
		if err != nil {
			logger.Default.Errorf("[Client] - readPump parse message for session: %v with err: %v", c.player.ID, err)
			msg, _ := messages.GenerateGenericMessage("error", "Invalid message format."+err.Error())
			c.send <- msg
			continue
		}
		if parsedmessage.Command == "ping" {
			msg := messages.MessageSimple{
				Command: "pong",
			}
			msgBytes, _ := json.Marshal(msg)
			c.send <- msgBytes
			continue
		}

		// we will check if the user session is stil valid.
		_, err = c.hub.redis.GetSessionByID(c.player.ID)
		if err != nil {
			logger.Default.Errorf("[Client] - readPump - session no longer exists, clossing player con with session id: %v, with error: %v", c.player.ID, err)
			break
		}

		// we will update out player object, if something is wrong with the update we will exit our loop.
		err = c.UpdatePlayerDataFromRedis()
		if err != nil {
			logger.Default.Errorf("[Client] - readPump - failed to update player data from redis with session id: %v, with error: %v", c.player.ID, err)
			break
		}
		go RouteMessages(parsedmessage, c, c.hub.redis)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		logger.Default.Infof("[Client] - readPump defer for session: %v", c.player.ID)
		c.cancel()
		c.hub.unregister <- c
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				logger.Default.Infof("[Client] - writePump - the hub closed the channel for session: %v", c.player.ID)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			//log.Print(message)
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logger.Default.Warnf("[Client] - writePump - failed getting message for session: %v, with message: %v", c.player.ID, err)
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				logger.Default.Warnf("[Client] - writePump - faileed closing writer for session: %v, with message: %v", c.player.ID, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Default.Info("[Client] - writePump - failed writing ping message for session: %v, with message: %v", c.player.ID, err)
				return
			}
		}
	}
}

func (c *Client) redisSubscribe(ready chan bool) {
	channel := redisdb.GetPlayerPubSubChannel(*c.player)
	pubsub := c.hub.redis.Client.Subscribe(c.ctx, channel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	ready <- true

	for {
		select {
		case msg := <-ch:
			if msg == nil {
				return
			}
			if strings.HasPrefix(msg.Payload, "disconnect:") {
				targetID := strings.TrimPrefix(msg.Payload, "disconnect:")
				if targetID == c.player.ID {
					logger.Default.Info("[Client] - redisSubscribe - received internal disconnected signal for session: %v, closing conn.", c.player.ID)
					c.conn.Close() // or any cleanup logic
					return
				}
			}
			c.send <- []byte(msg.Payload)
		case <-c.ctx.Done():
			return
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {

	// This makes a preliminar check to make sure the credentials are valid and the session exists.
	ok, session, err := AuthValid(w, r, hub.redis)
	if !ok {
		logger.Default.Warnf("[Client] - serveWs - auth failed with err: %v", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Now we will create our player, and check if there is any existing player with the same session.
	player, wasdisconnectedInGame, wasDisconnectedInQueue, err := CreatePlayer(hub.redis, session)
	if err != nil {
		logger.Default.Errorf("[Client] - serveWs - error creating player for session: %v, with err: %v", session.ID, err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	balance, err := FetchWalletBallance(session, hub.redis)
	if err != nil {
		logger.Default.Errorf("[Client] - serveWs - error fetching wallet for session: %v, with err: %v", session.ID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		hub.redis.RemovePlayer(player.ID)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Default.Errorf("[Client] - serveWs - error upgrading conn for session: %v, with err: %v", session.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		hub.redis.RemovePlayer(player.ID)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		hub: hub, conn: conn, player: player,
		send: make(chan []byte, 256),
		ctx:  ctx, cancel: cancel,
	}
	client.hub.register <- client

	// Subscription to redis pub sub.
	subscriptionReady := make(chan bool)
	go client.redisSubscribe(subscriptionReady)
	<-subscriptionReady // Wait for the subscription to be ready

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()

	msg, err := messages.GenerateConnectedMessage(*player, balance)
	if err != nil {
		logger.Default.Errorf("[Client] - serveWs - failled to generate connected message for session: %v, with err: %v", session.ID, err)
		client.CloseConnection()
		return
	}
	client.send <- msg

	// Now that our player has subscribbed to our stuff, we will notify the gameworker of the reconnect.
	if wasdisconnectedInGame {
		hub.redis.RPush("reconnect_game", player)
	}

	// The queue handling of a disconnected player also has to be done...
	// Its separated to try and not break the game reconect.
	if wasDisconnectedInQueue {
		hub.redis.DeleteDisconnectedInQueuePlayerData(player.ID)
		if player.Status == models.StatusInQueue {
			msg, err := messages.GenerateQueueConfirmationMessage(true)
			if err != nil {
				logger.Default.Errorf("[Client] - serveWs - failled to generate connected message when was DisconnectedInQueue for session: %v, with err: %v", session.ID, err)
				client.CloseConnection()
				return
			}
			client.send <- msg
		}
		if player.Status == models.StatusInRoom || player.Status == models.StatusInRoomReady {
			hub.redis.PublishToRoomPubSub(player.RoomID, "player_reconnect:"+player.ID)
		}
	}

}

func (c *Client) UpdatePlayerDataFromRedis() error {
	playerData, err := c.hub.redis.GetPlayer(string(c.player.ID))
	if err != nil {
		return fmt.Errorf("[UpdatePlayerDataFromRedis] - Failed to update player data from redis!: Player: %s", c.player.ID)
	}
	c.player.Currency = playerData.Currency
	c.player.Status = playerData.Status
	c.player.SelectedBet = playerData.SelectedBet
	c.player.RoomID = playerData.RoomID
	c.player.GameID = playerData.GameID
	return nil
}
