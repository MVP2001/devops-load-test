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
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	channels map[string]bool
	mu       sync.RWMutex
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	channels   map[string]map[*Client]bool
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		channels:   make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected. Total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				for channel, clients := range h.channels {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.channels, channel)
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := make(map[*Client]bool)
			for client := range h.clients {
				clients[client] = true
			}
			h.mu.RUnlock()
			
			for client := range clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					h.unregister <- client
				}
			}
		}
	}
}

func (h *Hub) BroadcastToChannel(channel string, payload interface{}) {
	message, err := json.Marshal(map[string]interface{}{
		"channel": channel,
		"data":    payload,
		"time":    time.Now().Unix(),
	})
	if err != nil {
		return
	}

	h.mu.RLock()
	clients, ok := h.channels[channel]
	h.mu.RUnlock()

	if !ok {
		return
	}

	for client := range clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			h.unregister <- client
		}
	}
}

func (h *Hub) Broadcast(payload interface{}) {
	message, err := json.Marshal(map[string]interface{}{
		"data": payload,
		"time": time.Now().Unix(),
	})
	if err != nil {
		return
	}
	h.broadcast <- message
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		channels: make(map[string]bool),
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			if action, ok := msg["action"].(string); ok {
				switch action {
				case "subscribe":
					if channel, ok := msg["channel"].(string); ok {
						c.subscribe(channel)
					}
				case "unsubscribe":
					if channel, ok := msg["channel"].(string); ok {
						c.unsubscribe(channel)
					}
				}
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
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

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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

func (c *Client) subscribe(channel string) {
	c.mu.Lock()
	c.channels[channel] = true
	c.mu.Unlock()

	c.hub.mu.Lock()
	if c.hub.channels[channel] == nil {
		c.hub.channels[channel] = make(map[*Client]bool)
	}
	c.hub.channels[channel][c] = true
	c.hub.mu.Unlock()
}

func (c *Client) unsubscribe(channel string) {
	c.mu.Lock()
	delete(c.channels, channel)
	c.mu.Unlock()

	c.hub.mu.Lock()
	if clients, ok := c.hub.channels[channel]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(c.hub.channels, channel)
		}
	}
	c.hub.mu.Unlock()
}
