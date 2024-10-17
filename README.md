Ah, você está certo! As constantes e a função `On` estavam no código original, mas acabei não as incluindo claramente na documentação. Vou corrigir isso agora!

Aqui está a versão atualizada do `README.md`:

---

# Huginn WebSocket Server

**Huginn** is a WebSocket server built in Go that simplifies managing WebSocket connections with optional MongoDB persistence. It provides features like client management, event-based message broadcasting, dynamic metadata handling, and graceful shutdown.

---

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [API Reference](#api-reference)
  - [Huginn Methods](#huginn-methods)
  - [Emit Constants](#emit-constants)
- [Event Handling](#event-handling)
- [Examples](#examples)
- [Shutdown](#shutdown)
- [License](#license)

---

## Features

- **WebSocket Client Management**: Add, remove, search, and update clients dynamically.
- **Broadcast Modes**: Emit messages to all clients, specific clients, or exclude certain clients.
- **MongoDB Persistence** (Optional): Store and manage client metadata.
- **Graceful Shutdown**: Close all active WebSocket connections safely.
- **Event-Driven Architecture**: Handle WebSocket events with custom handlers.

---

## Installation

First, ensure you have Go installed. Clone the repository and install the dependencies:

```bash
git clone https://github.com/IsaqueGeraldo/huginn.git
cd huginn
go mod tidy
```

Make sure to install MongoDB locally or use a MongoDB cloud instance if you plan to enable persistence.

---

## Usage

```go
package main

import (
	"log"
	"net/http"
	"github.com/IsaqueGeraldo/huginn"
	"github.com/gorilla/websocket"
)

func main() {
	// Initialize Huginn with MongoDB (optional)
	h := huginn.NewHuginn("mongodb://localhost:27017")

	// Register an event handler for "message" events
	h.On("message", func(msg huginn.Message, conn *websocket.Conn) error {
		log.Printf("Received message: %v", msg.Data)
		return nil
	})

	http.HandleFunc("/ws", h.Server)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Run the server:

```bash
go run main.go
```

---

## API Reference

### Huginn Methods

#### `NewHuginn(mongoURI ...string) *Huginn`

Creates a new instance of the Huginn WebSocket server.

- **Parameters**:  
  `mongoURI`: Optional MongoDB connection string.
- **Returns**: A new `Huginn` instance.

#### `On(event string, handler EventHandler)`

Registers a custom handler for a specific event.

- **Parameters**:
  - `event`: Name of the event.
  - `handler`: A function that takes a `Message` and `*websocket.Conn` as parameters and returns an error.

Example:

```go
h.On("chat", func(msg huginn.Message, conn *websocket.Conn) error {
    log.Printf("Chat event received: %s", msg.Data)
    return nil
})
```

#### `AddClient(conn *websocket.Conn) string`

Adds a new WebSocket client and returns the generated client UUID.

#### `RemoveClient(clientID string)`

Removes a client by its UUID and closes the connection.

#### `SearchClient(key string, value interface{}) (*Client, error)`

Searches for a client by a key-value pair in the metadata.

#### `UpdateMetadata(clientID, key string, val interface{})`

Updates or adds metadata for a specific client.

#### `RemoveMetadata(clientID, key string)`

Removes a key-value pair from a client’s metadata.

#### `Emit(message Message, mode int, clients []string)`

Broadcasts a message to clients based on the provided mode.

---

### Emit Constants

Huginn provides three modes for emitting messages:

```go
const (
    EmitAll = iota     // Send to all connected clients
    EmitOnly           // Send only to specific clients
    EmitExcept         // Send to all clients except specified ones
)
```

- **EmitAll**: Sends the message to all connected clients.
- **EmitOnly**: Sends the message only to the specified clients.
- **EmitExcept**: Sends the message to all clients except those listed.

---

## Event Handling

Huginn uses an event-driven model. You can register custom handlers for specific events using the `On` method.

When a client sends a message with an `event` field matching the registered event, the corresponding handler is executed.

Example of a message:

```json
{
  "event": "message",
  "data": {
    "content": "Hello, world!",
    "timestamp": "2024-10-17T12:00:00Z"
  }
}
```

---

## Examples

### Sending a Message to a Client

```go
msg := huginn.Message{Event: "message", Data: "Hello!"}
h.Emit(msg, huginn.EmitOnly, []string{"client-uuid-123"})
```

### Broadcasting to All Clients Except Specific Ones

```go
msg := huginn.Message{Event: "notification", Data: "Server update!"}
h.Emit(msg, huginn.EmitExcept, []string{"client-uuid-123"})
```

---

## Shutdown

Ensure a clean shutdown of the server by calling the `Shutdown` method:

```go
defer h.Shutdown()
```

This will close all active WebSocket connections and clean up MongoDB resources if enabled.

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.
