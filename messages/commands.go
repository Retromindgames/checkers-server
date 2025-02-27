package messages

type CommandType string

const (
	ServerCommand    CommandType = "server"
	ClientCommand    CommandType = "client"
	BroadcastCommand CommandType = "broadcast"
)

type CommandInfo struct {
	Type CommandType
}

var validCommands = map[string]CommandInfo{
	"queue":       {Type: ClientCommand}, // This adds the player to a queue.
	"ready_queue": {Type: ClientCommand}, // This allows the player to issue a ready when in a room.
	"leave_queue": {Type: ClientCommand}, // This allows the player to leave the queue.
	"leave_room":  {Type: ClientCommand}, // This allows the player to leave the room.
	"join_room":   {Type: ClientCommand}, // DEPRECATED
	"create_room": {Type: ClientCommand}, // DEPRECATED

	"move_piece": {Type: ClientCommand},

	"connected":          {Type: ServerCommand},
	"queue_confirmation": {Type: ServerCommand}, // This confirms that the player was placed in Queue.
	"room_created":       {Type: ServerCommand},
	"paired":             {Type: ServerCommand}, // This lets players know they were paired.
	"opponent_ready":     {Type: ServerCommand}, // This lets the player know if the opponent is ready.
	"opponent_left_room": {Type: ServerCommand},
	"game_start":         {Type: ServerCommand},
	"game_timer":         {Type: ServerCommand},
	"game_over":          {Type: ServerCommand},
	"turn_switch":        {Type: ServerCommand},
	"balance_update":     {Type: ServerCommand},

	"game_info": {Type: BroadcastCommand},
}
