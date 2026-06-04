package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/embedded-ops-platform/backend/pkg/logger"
	"github.com/gorilla/websocket"
)

// Message types for WebSocket communication.
const (
	MsgTypeDeviceOnline  = "device_online"
	MsgTypeDeviceOffline = "device_offline"
	MsgTypeMetricsUpdate = "metrics_update"
	MsgTypeCommandResult = "command_result"
	MsgTypeAlert         = "alert"
	MsgTypeCommand       = "command"
)

// Message represents a WebSocket message.
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Hub maintains a set of active WebSocket connections for the frontend.
// It is designed for high concurrency with a fan-out pattern.
type Hub struct {
	mu           sync.RWMutex
	clients      map[*Client]bool
	broadcast    chan Message
	registerCh   chan *Client
	unregisterCh chan *Client
	done         chan struct{}
}

// Client represents a single WebSocket connection.
type Client struct {
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
	UserID   string
	userRole string
	mu       sync.Mutex
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		broadcast:    make(chan Message, 256),
		registerCh:   make(chan *Client, 64),
		unregisterCh: make(chan *Client, 64),
		done:         make(chan struct{}),
	}
}

// Run starts the hub's event loop.
func (h *Hub) Run() {
	logger.Info("WebSocket hub started")
	for {
		select {
		case client := <-h.registerCh:
			h.mu.Lock()
			h.clients[client] = true
			count := len(h.clients)
			h.mu.Unlock()
			logger.Info("WebSocket client connected",
				logger.String("user_id", client.UserID),
				logger.Int("total_clients", count))

		case client := <-h.unregisterCh:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			count := len(h.clients)
			h.mu.Unlock()
			logger.Info("WebSocket client disconnected",
				logger.String("user_id", client.UserID),
				logger.Int("total_clients", count))

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg)
			if err != nil {
				logger.Warn("Failed to marshal broadcast message", logger.ErrField(err))
				continue
			}
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- data:
				default:
					// Client send buffer full, drop client
					go func(c *Client) {
						h.unregisterCh <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		case <-h.done:
			return
		}
	}
}

// Stop gracefully stops the hub.
func (h *Hub) Stop() {
	close(h.done)
	h.mu.Lock()
	for client := range h.clients {
		client.Conn.Close()
	}
	h.mu.Unlock()
}

// Broadcast sends a message to all connected frontend clients.
func (h *Hub) Broadcast(msgType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		logger.Warn("Failed to marshal broadcast payload", logger.ErrField(err))
		return
	}
	msg := Message{
		Type:    msgType,
		Payload: data,
	}
	select {
	case h.broadcast <- msg:
	default:
		logger.Warn("Broadcast channel full, dropping message")
	}
}

// Register adds a new client.
func (h *Hub) Register(client *Client) {
	h.registerCh <- client
}

// Unregister removes a client.
func (h *Hub) Unregister(client *Client) {
	h.unregisterCh <- client
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// WritePump pumps messages from the hub to the WebSocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.mu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := c.Conn.WriteMessage(websocket.TextMessage, message)
			c.mu.Unlock()
			if err != nil {
				logger.Warn("WebSocket write error",
					logger.String("user_id", c.UserID),
					logger.ErrField(err))
				return
			}
		case <-ticker.C:
			c.mu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := c.Conn.WriteMessage(websocket.PingMessage, nil)
			c.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(4096)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_ = time.Now()
		c.mu.Unlock()
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Warn("WebSocket read error", logger.ErrField(err))
			}
			break
		}
	}
}