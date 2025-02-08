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

### Example 3: Custom value (WIP)

Used to send more complex data, needs to be worked on.

```json
{
  "command": "custom_command",
  "value": { "key": "value" }
}
```
