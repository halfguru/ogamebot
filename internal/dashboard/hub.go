package dashboard

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 54 * time.Second
	// Maximum number of WebSocket clients to prevent DoS (T-05-02).
	maxClients = 10
)

// Upgrader upgrades HTTP connections to WebSocket.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin returns true — CORS is handled at the server middleware level.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub maintains the set of active WebSocket clients and broadcasts messages.
type Hub struct {
	// Registered clients.
	clients map[*client]bool
	// Inbound messages from handlers to broadcast.
	broadcast chan []byte
	// Register requests from clients.
	register chan *client
	// Unregister requests from clients.
	unregister chan *client
	// Mutex for thread-safe client count checks.
	mu sync.Mutex
	// Logger for hub events.
	log *slog.Logger
}

// client represents a single WebSocket connection.
type client struct {
	hub  *Hub
	conn *websocket.Conn
	// Buffered channel of outbound messages.
	send chan []byte
}

// NewHub creates a new Hub instance.
func NewHub(log *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *client),
		unregister: make(chan *client),
		log:        log.With("component", "dashboard-hub"),
	}
}

// Run starts the hub's event loop. Blocks until context is cancelled.
func (h *Hub) Run(ctx context.Context) {
	h.log.Info("WebSocket hub started")
	for {
		select {
		case <-ctx.Done():
			h.log.Info("WebSocket hub stopped")
			// Close all client connections
			for c := range h.clients {
				close(c.send)
			}
			return
		case c := <-h.register:
			h.mu.Lock()
			if len(h.clients) >= maxClients {
				h.mu.Unlock()
				h.log.Warn("Max WebSocket clients reached, rejecting connection", "maxClients", maxClients)
				close(c.send)
				return
			}
			h.clients[c] = true
			count := len(h.clients)
			h.mu.Unlock()
			h.log.Info("WebSocket client connected", "totalClients", count)
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			count := len(h.clients)
			h.mu.Unlock()
			h.log.Info("WebSocket client disconnected", "totalClients", count)
		case message := <-h.broadcast:
			h.mu.Lock()
			for c := range h.clients {
				select {
				case c.send <- message:
				default:
					// Buffer full — drop the client
					delete(h.clients, c)
					close(c.send)
					h.log.Warn("Dropped slow WebSocket client")
				}
			}
			h.mu.Unlock()
		}
	}
}

// Broadcast sends a typed message to all connected WebSocket clients.
func (h *Hub) Broadcast(msgType string, data interface{}) {
	msg := WSMessage{
		Type: msgType,
		Data: data,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		h.log.Error("Failed to marshal broadcast message", "type", msgType, "error", err)
		return
	}
	select {
	case h.broadcast <- payload:
	default:
		h.log.Warn("Broadcast channel full, dropping message", "type", msgType)
	}
}

// serveWS handles WebSocket upgrade requests.
func (h *Hub) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("WebSocket upgrade failed", "error", err)
		return
	}

	c := &client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}
	h.register <- c

	// Start pumps in goroutines
	go c.writePump()
	go c.readPump()
}

// readPump reads messages from the WebSocket connection.
// It detects disconnections and unregisters the client.
func (c *client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.log.Error("WebSocket read error", "error", err)
			}
			break
		}
	}
}

// writePump writes messages from the hub to the WebSocket connection.
// It also sends periodic pings to keep the connection alive.
func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
