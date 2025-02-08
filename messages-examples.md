# WebSocket Messages

---

## Cliente

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

### Example 2: Leave Queue

This is used to leave the queue.

```json
{
  "command": "leave_queue"
}
```

```json
{
  "command": "join_queue",
  "value": 0.5
}
```

### Example 3: Send Message (NOT IMPLEMENTED)

Doesnt do anything.

```json
{
  "command": "send_message",
  "value": "msg"
}
```

### Example 4: Custom value (WIP)

Used to send more complex data, needs to be worked on.

```json
{
  "command": "custom_command",
  "value": { "key": "value" }
}
```

---

# Server

### Example 1: Custom value (WIP)

When a patch starts, the value represents the color, 1 = black, 0 = white.

```json
{
  "command": "paired",
  "value": 1
}
```
