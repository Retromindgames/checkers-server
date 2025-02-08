package message

import (
	"checkers-server/core"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type Message struct {
	Command string          `json:"command"`
	Value   json.RawMessage `json:"value,omitempty"` 
}

func ParseMessage(msg []byte, conn *websocket.Conn) (*Message, error) {
	var message Message
	if err := json.Unmarshal(msg, &message); err != nil {
		return nil, fmt.Errorf("invalid message format: %v", err)
	}

	// Parse based on command, this checks if the value of the command is the expected.
	// this way, the message is only hald decoded, and when we need the rest, we are pretty sure its the right type (hope).
	switch message.Command {
		case "leave_queue":
			return &message, nil	
			
		case "join_queue":
			var value float64
			if err := json.Unmarshal(message.Value, &value); err != nil {
				return nil, fmt.Errorf("invalid value format for join_queue: %v", err)
			}
			message.Value = json.RawMessage(fmt.Sprintf("%v", value)) 							// Store it back as raw JSON

		case "send_message":
			// not sure if we will need this.
			var value string
			if err := json.Unmarshal(message.Value, &value); err != nil {
				return nil, fmt.Errorf("invalid value format for send_message: %v", err)
			}
			message.Value = json.RawMessage(fmt.Sprintf("\"%s\"", value)) 		

		case "custom_command":
			// Expected to be an object or array
			var value map[string]interface{}
			if err := json.Unmarshal(message.Value, &value); err != nil {
				// unmarshalling as an array if not an object, maybe redundant.
				var valueArray []interface{}
				if err := json.Unmarshal(message.Value, &valueArray); err != nil {
					return nil, fmt.Errorf("invalid value format for custom_command: %v", err)
				}
				message.Value = json.RawMessage(fmt.Sprintf("%v", valueArray)) 					
			} else {
				message.Value = json.RawMessage(fmt.Sprintf("%v", value)) 						
			}
		default:
	}

	return &message, nil
}

func GenerateConnectedMessage(player *core.Player) (string, error) {
	response := struct {
		Command   string  `json:"command"`
		Value     struct {
			PlayerName string  `json:"player_name"`
			Money      float64 `json:"money"`
		} `json:"value"`
	}{
		Command: "connected",
	}
	response.Value.PlayerName = "Player_" + player.Name
	response.Value.Money = player.Money
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return "", err
	}
	return string(jsonResponse), nil
}

func GeneratePairedMessage(player1, player2 *core.Player, color int) string {
	return fmt.Sprintf(`{
		"command": "paired",
		"value": {
			"color": %d,
			"opponent": "%s"
		}
	}`, color, player2.Name)
}