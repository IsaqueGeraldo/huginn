package huginn

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Metadata stores client-specific data.
type Metadata struct {
	UUID string                 `bson:"uuid"`
	Data map[string]interface{} `bson:"data"`
}

var upgrader = websocket.Upgrader{
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Printf("[huginn]: WebSocket Upgrade error: %v\n", reason)
	},
	CheckOrigin: func(r *http.Request) bool {
		// Adjust this to allow only trusted origins
		return true
	},
}

// Client represents an individual WebSocket connection.
type Client struct {
	Conn     *websocket.Conn
	UUID     string
	Metadata Metadata
}

// Message defines the structure for WebSocket events.
type Message struct {
	Event string      `json:"event,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

// EventHandler defines the function signature for event handling.
type EventHandler func(msg Message, conn *websocket.Conn) error

// Huginn manages WebSocket clients, events, and optional persistence.
type Huginn struct {
	Clients  sync.Map // Use sync.Map instead of manual mutex locking
	Events   map[string]EventHandler
	Upgrader *websocket.Upgrader
	DB       *mongo.Collection // Optional MongoDB collection
}

// NewHuginn creates a new Huginn instance. MongoDB connection is optional.
func NewHuginn(mongoURI ...string) *Huginn {
	var collection *mongo.Collection

	if len(mongoURI) > 0 && mongoURI[0] != "" {
		col, err := connectMongo(mongoURI[0])
		if err != nil {
			log.Printf("[huginn]: Error connecting to MongoDB: %v", err)
		} else {
			collection = col
		}
	}

	return &Huginn{
		Events:   make(map[string]EventHandler),
		Upgrader: &upgrader,
		DB:       collection,
	}
}

// connectMongo handles MongoDB connection.
func connectMongo(mongoURI string) (*mongo.Collection, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}
	return client.Database("huginn").Collection("clients"), nil
}

// AddClient adds a new WebSocket client and stores it with optional persistence.
func (h *Huginn) AddClient(conn *websocket.Conn) string {
	clientID := uuid.New().String()
	metadata := Metadata{UUID: clientID, Data: map[string]interface{}{}}
	client := &Client{Conn: conn, UUID: clientID, Metadata: metadata}

	h.Clients.Store(clientID, client)

	if h.DB != nil {
		if _, err := h.DB.InsertOne(context.TODO(), metadata); err != nil {
			log.Printf("[huginn]: Error storing client in MongoDB: %v", err)
		}
	}

	return clientID
}

// RemoveClient removes a client and deletes its data if MongoDB is enabled.
func (h *Huginn) RemoveClient(clientID string) {
	if value, ok := h.Clients.LoadAndDelete(clientID); ok {
		client := value.(*Client)
		_ = client.Conn.Close() // Ignore error if already closed
	}

	if h.DB != nil {
		if _, err := h.DB.DeleteOne(context.TODO(), bson.M{"uuid": clientID}); err != nil {
			log.Printf("[huginn]: Error removing client from MongoDB: %v", err)
		}
	}
}

// SearchClient searches for a client by a specific key and value in the metadata.
func (h *Huginn) SearchClient(key string, value interface{}) (*Client, error) {
	var result Metadata
	filter := bson.M{"data." + key: value}
	err := h.DB.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return nil, err
	}

	if client, ok := h.Clients.Load(result.UUID); ok {
		return client.(*Client), nil
	}
	return nil, fmt.Errorf("client not found in active connections")
}

// UpdateMetadata adds or updates a key-value pair in a client's metadata.
func (h *Huginn) UpdateMetadata(clientID, key string, val interface{}) {
	if value, ok := h.Clients.Load(clientID); ok {
		client := value.(*Client)
		client.Metadata.Data[key] = val

		if h.DB != nil {
			_, err := h.DB.UpdateOne(
				context.TODO(),
				bson.M{"uuid": clientID},
				bson.M{"$set": bson.M{"data." + key: val}},
			)
			if err != nil {
				log.Printf("[huginn]: Error updating metadata in MongoDB: %v", err)
			}
		}
	}
}

// RemoveMetadata removes a key from the client's metadata.
func (h *Huginn) RemoveMetadata(clientID, key string) {
	if value, ok := h.Clients.Load(clientID); ok {
		client := value.(*Client)
		delete(client.Metadata.Data, key)

		if h.DB != nil {
			_, err := h.DB.UpdateOne(
				context.TODO(),
				bson.M{"uuid": clientID},
				bson.M{"$unset": bson.M{"data." + key: ""}},
			)
			if err != nil {
				log.Printf("[huginn]: Error removing metadata from MongoDB: %v", err)
			}
		}
	}
}

// sendMessage sends a message to a specific WebSocket connection.
func (h *Huginn) sendMessage(clientID string, msg Message) {
	if value, ok := h.Clients.Load(clientID); ok {
		client := value.(*Client)
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("[huginn]: Error marshalling message: %v", err)
			return
		}

		client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[huginn]: Error sending message: %v", err)
			h.RemoveClient(clientID)
		}
	}
}

// On registers a handler for a specific event.
func (h *Huginn) On(event string, handler EventHandler) {
	h.Events[event] = handler
}

// Emit sends a message to clients based on the selected mode.
const (
	EmitAll = iota
	EmitOnly
	EmitExcept
)

// Emit sends a message to clients based on the selected mode.
func (h *Huginn) Emit(message Message, mode int, clients []string) {
	switch mode {
	case EmitAll:
		h.Clients.Range(func(_, value interface{}) bool {
			client := value.(*Client)
			h.sendMessage(client.UUID, message)
			return true
		})
	case EmitOnly:
		clientSet := make(map[string]struct{}, len(clients))
		for _, id := range clients {
			clientSet[id] = struct{}{}
		}
		h.Clients.Range(func(key, value interface{}) bool {
			if _, exists := clientSet[key.(string)]; exists {
				h.sendMessage(key.(string), message)
			}
			return true
		})
	case EmitExcept:
		excludeSet := make(map[string]struct{}, len(clients))
		for _, id := range clients {
			excludeSet[id] = struct{}{}
		}
		h.Clients.Range(func(key, value interface{}) bool {
			if _, excluded := excludeSet[key.(string)]; !excluded {
				h.sendMessage(key.(string), message)
			}
			return true
		})
	}
}

// Shutdown gracefully closes all client connections and cleans up resources.
func (h *Huginn) Shutdown() {
	h.Clients.Range(func(_, value interface{}) bool {
		client := value.(*Client)
		_ = client.Conn.Close()
		return true
	})

	if h.DB != nil {
		if err := h.DB.Database().Client().Disconnect(context.TODO()); err != nil {
			log.Printf("[huginn]: Error disconnecting from MongoDB: %v", err)
		}
	}
}

// Server handles incoming WebSocket connections.
func (h *Huginn) Server(w http.ResponseWriter, r *http.Request) {
	conn, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[huginn]: WebSocket upgrade error: %v", err)
		return
	}

	clientID := h.AddClient(conn)
	log.Printf("[huginn]: New client connected: %s", clientID)

	defer h.RemoveClient(clientID)

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("[huginn]: Error reading message: %v", err)
			break
		}

		if handler, exists := h.Events[msg.Event]; exists {
			if err := handler(msg, conn); err != nil {
				log.Printf("[huginn]: Error handling event '%s': %v", msg.Event, err)
			}
		} else {
			_ = conn.WriteJSON(Message{Event: "error", Data: "Event not found: " + msg.Event})
		}
	}
}
