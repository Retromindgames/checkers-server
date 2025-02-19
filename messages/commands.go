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
	"queue":              {Type: ClientCommand},
	"queue_confirmation": {Type: ClientCommand},
	"create_room":        {Type: ClientCommand},
	"ready_queue":        {Type: ClientCommand},
	"join_room":          {Type: ClientCommand},
	"leave_room":         {Type: ClientCommand},

	"connected":      {Type: ServerCommand},
	"room_created":   {Type: ServerCommand},
	"paired":         {Type: ServerCommand},
	"opponent_ready": {Type: ServerCommand},

	"game_info": {Type: BroadcastCommand},
}
