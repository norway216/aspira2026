package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"lan-monitor/internal/models"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type DeviceStatusUpdate struct {
	DeviceID  uint   `json:"device_id"`
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	Hostname  string `json:"hostname"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	topics map[string]bool
	mu     sync.Mutex
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	hub := &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go hub.run()
	return hub
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastDeviceStatus sends device status update to all clients
func (h *Hub) BroadcastDeviceStatus(updates []DeviceStatusUpdate) {
	msg := WSMessage{
		Type: "device_status",
		Data: updates,
	}
	data, _ := json.Marshal(msg)
	h.broadcast <- data
}

// BroadcastAlert sends an alert to all clients
func (h *Hub) BroadcastAlert(alert models.Alert) {
	msg := WSMessage{
		Type: "alert",
		Data: alert,
	}
	data, _ := json.Marshal(msg)
	h.broadcast <- data
}

// BroadcastTraffic sends traffic data
func (h *Hub) BroadcastTraffic(deviceID uint, traffic interface{}) {
	msg := WSMessage{
		Type: "traffic",
		Data: map[string]interface{}{
			"device_id": deviceID,
			"traffic":   traffic,
		},
	}
	data, _ := json.Marshal(msg)
	h.broadcast <- data
}

// BroadcastScanProgress sends scan progress
func (h *Hub) BroadcastScanProgress(taskID uint, status string, progress int) {
	msg := WSMessage{
		Type: "scan_progress",
		Data: map[string]interface{}{
			"task_id":  taskID,
			"status":   status,
			"progress": progress,
		},
	}
	data, _ := json.Marshal(msg)
	h.broadcast <- data
}

// HandleWebSocket handles a new WebSocket connection
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 256),
		topics: make(map[string]bool),
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(4096)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		// Handle client messages (subscribe/unsubscribe topics)
		var msg map[string]interface{}
		if json.Unmarshal(message, &msg) == nil {
			if topic, ok := msg["subscribe"].(string); ok {
				c.mu.Lock()
				c.topics[topic] = true
				c.mu.Unlock()
			}
			if topic, ok := msg["unsubscribe"].(string); ok {
				c.mu.Lock()
				delete(c.topics, topic)
				c.mu.Unlock()
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// GetConnectedClients returns the number of connected clients
func (h *Hub) GetConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
