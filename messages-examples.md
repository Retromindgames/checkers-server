# WebSocket Messages

### Example 1: Join Queue

This is used to join the queue with a bid value.

```json
{
  "command": "join_queue",
  "value": 2
}
```

```json
{
  "command": "join_queue",
  "value": 0.5
}
```

### Example 2: Send Message (NOT IMPLEMENTED)

Doesnt do anything.

```json
{
  "command": "send_message",
  "value": "msg"
}
```

### Example 3: Join Queue

This message is used to join the queue with a bid value.

```json
{
  "command": "custom_command",
  "value": { "key": "value" }
}
```
