package messages

import (
	"checkers-server/models"
	"encoding/json"
	"fmt"
)

type Message struct {
	Command string          `json:"command"`
	Value   json.RawMessage `json:"value,omitempty"`
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type MovePieceMessage struct {
	Command string `json:"command"`
	Value   struct {
		PreviousPosition Position   `json:"previous_position"`
		NewPosition      Position   `json:"new_position"`
		KilledPieces     []Position `json:"killed_pieces"`
	} `json:"value"`
}

func ParseMessage(msg []byte) (*Message, error) {
	var message Message
	if err := json.Unmarshal(msg, &message); err != nil {
		return nil, fmt.Errorf("invalid message format: %v", err)
	}

	// Parse based on command, this checks if the value of the command is the expected.
	// this way, the message is only hald decoded, and when we need the rest, we are pretty sure its the right type (hope).
	switch message.Command {
		case "leave_queue":
			return &message, nil

		case "room_created":
			var value string
			if err := json.Unmarshal(message.Value, &value); err != nil {
				return nil, fmt.Errorf("invalid value format for room_created: %v", err)
			}
			message.Value = json.RawMessage(message.Value)

		case "create_room":
			var value float64
			if err := json.Unmarshal(message.Value, &value); err != nil {
				return nil, fmt.Errorf("invalid value format for create_room: %v", err)
			}
			message.Value = json.RawMessage(message.Value) 							// Store it back as raw JSON

		case "join_room":
			var value float64
			if err := json.Unmarshal(message.Value, &value); err != nil {
				return nil, fmt.Errorf("invalid value format for join_room: %v", err)
			}
			message.Value = json.RawMessage(message.Value) 							// Store it back as raw JSON

		case "send_message":
			// not sure if we will need this.
			var value string
			if err := json.Unmarshal(message.Value, &value); err != nil {
				return nil, fmt.Errorf("invalid value format for send_message: %v", err)
			}
			message.Value = json.RawMessage(fmt.Sprintf("\"%s\"", value))

		case "move_piece":
			var value MovePieceMessage
			if err := json.Unmarshal(message.Value, &value.Value); err != nil {
				return nil, fmt.Errorf("invalid value format for move_piece: %v", err)
			}
			message.Value, _ = json.Marshal(message)											// ! For convinience we store the whole message in the value. To sendo it arround.

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

func GenerateConnectedMessage(player *models.Player) (string, error) {
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
	response.Value.Money = float64(player.CurrencyAmount)
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return "", err
	}
	return string(jsonResponse), nil
}

func GeneratePairedMessage(player1, player2 *models.Player, color int) string {
	return fmt.Sprintf(`{
		"command": "paired",
		"value": {
			"color": %d,
			"opponent": "%s"
		}
	}`, color, player2.Name)
}
