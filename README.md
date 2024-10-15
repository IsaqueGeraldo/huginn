Perfeito! Aqui está a documentação ajustada para usar o `go get`:

---

# Huginn Documentation

## Overview

Huginn is a WebSocket server written in Go, designed for managing connections, handling events, and storing client metadata. It offers MongoDB integration for persistence but can run without it. This guide will help you get started quickly.

---

## Table of Contents

1. [Installation](#installation)
2. [Configuration](#configuration)
3. [Core Concepts](#core-concepts)
4. [API Reference](#api-reference)
5. [Usage Examples](#usage-examples)
6. [Handling Events](#handling-events)
7. [Shutdown](#shutdown)
8. [Contributing](#contributing)

---

## Installation

1. **Add Huginn to your Go project:**

   ```bash
   go get github.com/IsaqueGeraldo/huginn
   ```

2. **Import Huginn in your code:**

   ```go
   import "github.com/IsaqueGeraldo/huginn"
   ```

3. **Ensure dependencies are up to date:**

   ```bash
   go mod tidy
   ```

---

## Configuration

Create a Huginn instance with optional MongoDB support:

```go
h := huginn.NewHuginn("your_mongo_uri_here") // MongoDB is optional
```

If you do not want to use MongoDB, just call:

```go
h := huginn.NewHuginn()
```

---

## Core Concepts

- **Client:** Represents a WebSocket connection with associated metadata.
- **Event:** An action sent over the WebSocket, with a handler that defines how to process it.
- **Metadata:** Stores additional data about a client.
- **Modes:** Used in message broadcasting (All, Only specific clients, or Except specific clients).

---

## API Reference

### Creating a Huginn Instance

```go
h := huginn.NewHuginn("mongo_uri_here") // Optional MongoDB URI
```

### Adding a Client

```go
clientID := h.AddClient(conn)
```

### Removing a Client

```go
h.RemoveClient(clientID)
```

### Searching for a Client by Key/Value

```go
client, err := h.SearchClient("key", "value")
if err != nil {
    log.Println("Client not found")
}
```

### Updating Metadata

```go
h.UpdateMetadata(clientID, "key", "value")
```

### Removing Metadata

```go
h.RemoveMetadata(clientID, "key")
```

### Emitting Events to Clients

```go
h.Emit(huginn.Message{Event: "test", Data: "hello"}, huginn.EmitAll, nil)
```

### Graceful Shutdown

```go
h.Shutdown()
```

---

## Usage Examples

### Example: Creating a Basic Server

```go
package main

import (
    "log"
    "net/http"
    "github.com/IsaqueGeraldo/huginn"
)

func main() {
    h := huginn.NewHuginn() // Without MongoDB

    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        conn, err := h.Upgrader.Upgrade(w, r, nil)
        if err != nil {
            log.Println("Upgrade error:", err)
            return
        }
        h.AddClient(conn)
    })

    log.Println("Server started on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

---

## Handling Events

You can register custom event handlers with the `On` method:

```go
h.On("greet", func(msg huginn.Message, conn *websocket.Conn) error {
    response := huginn.Message{Event: "response", Data: "Hello, client!"}
    return conn.WriteJSON(response)
})
```

---

## Shutdown

When the server is stopped, ensure all connections are closed gracefully:

```go
h.Shutdown()
```

---

## Contributing

Contributions are welcome! If you find a bug or want to propose a feature, please open an issue or submit a pull request on [GitHub](https://github.com/IsaqueGeraldo/huginn).

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
