package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	topics map[string]bool
	mu     sync.RWMutex
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
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
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
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

		var msg struct {
			Subscribe   string `json:"subscribe"`
			Unsubscribe string `json:"unsubscribe"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.mu.Lock()
		if msg.Subscribe != "" {
			c.topics[msg.Subscribe] = true
		}
		if msg.Unsubscribe != "" {
			delete(c.topics, msg.Unsubscribe)
		}
		c.mu.Unlock()
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

func (h *Hub) BroadcastJSON(msgType string, data interface{}) {
	msg, err := json.Marshal(map[string]interface{}{
		"type": msgType,
		"data": data,
	})
	if err != nil {
		log.Printf("Failed to marshal WS message: %v", err)
		return
	}
	h.broadcast <- msg
}

func (h *Hub) BroadcastTransactionUpdate(txn interface{}) {
	h.BroadcastJSON("transaction_update", txn)
}

func (h *Hub) BroadcastEngineHealth(health interface{}) {
	h.BroadcastJSON("engine_health", health)
}

func (h *Hub) BroadcastAlert(level, title, message string) {
	h.BroadcastJSON("alert", map[string]string{
		"level":   level,
		"title":   title,
		"message": message,
	})
}

func (h *Hub) BroadcastDashboardStats(stats interface{}) {
	h.BroadcastJSON("dashboard_stats", stats)
}

func (h *Hub) BroadcastTPSUpdate(tps float64) {
	h.BroadcastJSON("tps_update", map[string]float64{"tps": tps})
}

func (h *Hub) GetActiveConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

