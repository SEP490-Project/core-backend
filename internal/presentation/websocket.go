package presentation

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
	"core-backend/config"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocket message types
const (
	MessageTypeText         = "text"
	MessageTypeNotification = "notification"
	MessageTypeHeartbeat    = "heartbeat"
	MessageTypeError        = "error"
	MessageTypeAuth         = "auth"
	MessageTypeUserJoined   = "user_joined"
	MessageTypeUserLeft     = "user_left"
)

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    string      `json:"user_id,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	UserID   string
	Conn     *websocket.Conn
	Send     chan WebSocketMessage
	Hub      *Hub
	LastSeen time.Time
	mutex    sync.RWMutex
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan WebSocketMessage
	register   chan *Client
	unregister chan *Client
	userConns  map[string][]*Client // UserID -> []*Client
	mutex      sync.RWMutex
}

// WebSocketServer wraps the Hub and provides WebSocket functionality
type WebSocketServer struct {
	hub      *Hub
	upgrader websocket.Upgrader
	config   *config.WebSocketConfig
}

func NewWebSocketServer() *WebSocketServer {
	cfg := config.GetAppConfig().WebSocket

	upgrader := websocket.Upgrader{
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if len(cfg.AllowedOrigins) == 0 {
				return true
			}
			for _, allowedOrigin := range cfg.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					return true
				}
			}
			return false
		},
	}

	hub := &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan WebSocketMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		userConns:  make(map[string][]*Client),
	}

	return &WebSocketServer{
		hub:      hub,
		upgrader: upgrader,
		config:   &cfg,
	}
}

// Start starts the WebSocket hub
func (ws *WebSocketServer) Start(ctx context.Context) {
	zap.L().Info("Starting WebSocket server...")
	go ws.hub.run(ctx)
}

// HandleWebSocket handles WebSocket connections
func (ws *WebSocketServer) HandleWebSocket(c *gin.Context) {
	conn, err := ws.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zap.L().Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "Unauthorized"))
		conn.Close()
		return
	}

	client := &Client{
		ID:       generateClientID(),
		UserID:   userID.(string),
		Conn:     conn,
		Send:     make(chan WebSocketMessage, 256),
		Hub:      ws.hub,
		LastSeen: time.Now(),
	}

	ws.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()

	zap.L().Info("WebSocket client connected",
		zap.String("client_id", client.ID),
		zap.String("user_id", client.UserID))
}

// run starts the hub's main loop
func (h *Hub) run(ctx context.Context) {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			if h.userConns[client.UserID] == nil {
				h.userConns[client.UserID] = make([]*Client, 0)
			}
			h.userConns[client.UserID] = append(h.userConns[client.UserID], client)
			h.mutex.Unlock()

			// Send welcome message
			welcomeMsg := WebSocketMessage{
				Type:      MessageTypeNotification,
				Data:      map[string]string{"message": "Connected successfully"},
				Timestamp: time.Now(),
			}
			select {
			case client.Send <- welcomeMsg:
			default:
				close(client.Send)
				h.removeClient(client)
			}

		case client := <-h.unregister:
			h.removeClient(client)

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()

		case <-ticker.C:
			h.cleanupInactiveClients()

		case <-ctx.Done():
			zap.L().Info("WebSocket hub shutting down...")
			h.closeAllClients()
			return
		}
	}
}

// removeClient removes a client from the hub
func (h *Hub) removeClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.Send)

		// Remove from user connections
		if userConns, exists := h.userConns[client.UserID]; exists {
			for i, c := range userConns {
				if c.ID == client.ID {
					h.userConns[client.UserID] = append(userConns[:i], userConns[i+1:]...)
					break
				}
			}
			if len(h.userConns[client.UserID]) == 0 {
				delete(h.userConns, client.UserID)
			}
		}

		zap.L().Info("WebSocket client disconnected",
			zap.String("client_id", client.ID),
			zap.String("user_id", client.UserID))
	}
}

// cleanupInactiveClients removes clients that haven't been active
func (h *Hub) cleanupInactiveClients() {
	h.mutex.RLock()
	inactiveClients := make([]*Client, 0)
	for client := range h.clients {
		client.mutex.RLock()
		if time.Since(client.LastSeen) > 2*time.Minute {
			inactiveClients = append(inactiveClients, client)
		}
		client.mutex.RUnlock()
	}
	h.mutex.RUnlock()

	for _, client := range inactiveClients {
		h.unregister <- client
	}
}

// closeAllClients closes all client connections
func (h *Hub) closeAllClients() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for client := range h.clients {
		client.Conn.Close()
		close(client.Send)
	}
	h.clients = make(map[*Client]bool)
	h.userConns = make(map[string][]*Client)
}

// BroadcastToAll broadcasts a message to all connected clients
func (ws *WebSocketServer) BroadcastToAll(msgType string, data interface{}) {
	message := WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
	ws.hub.broadcast <- message
}

// BroadcastToUser broadcasts a message to all connections of a specific user
func (ws *WebSocketServer) BroadcastToUser(userID, msgType string, data interface{}) {
	ws.hub.mutex.RLock()
	userConns, exists := ws.hub.userConns[userID]
	if !exists {
		ws.hub.mutex.RUnlock()
		return
	}

	message := WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		UserID:    userID,
	}

	for _, client := range userConns {
		select {
		case client.Send <- message:
		default:
			// Client's send channel is blocked, remove it
			ws.hub.unregister <- client
		}
	}
	ws.hub.mutex.RUnlock()
}

// GetConnectedUsers returns a list of currently connected user IDs
func (ws *WebSocketServer) GetConnectedUsers() []string {
	ws.hub.mutex.RLock()
	defer ws.hub.mutex.RUnlock()

	users := make([]string, 0, len(ws.hub.userConns))
	for userID := range ws.hub.userConns {
		users = append(users, userID)
	}
	return users
}

// GetConnectionCount returns the total number of active connections
func (ws *WebSocketServer) GetConnectionCount() int {
	ws.hub.mutex.RLock()
	defer ws.hub.mutex.RUnlock()
	return len(ws.hub.clients)
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				zap.L().Error("Failed to write message", zap.Error(err))
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var message WebSocketMessage
		err := c.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				zap.L().Error("WebSocket error", zap.Error(err))
			}
			break
		}

		c.mutex.Lock()
		c.LastSeen = time.Now()
		c.mutex.Unlock()

		// Handle different message types
		c.handleMessage(message)
	}
}

// handleMessage processes incoming messages from clients
func (c *Client) handleMessage(message WebSocketMessage) {
	switch message.Type {
	case MessageTypeHeartbeat:
		// Respond to heartbeat
		response := WebSocketMessage{
			Type:      MessageTypeHeartbeat,
			Data:      map[string]string{"status": "alive"},
			Timestamp: time.Now(),
		}
		select {
		case c.Send <- response:
		default:
		}

	case MessageTypeText:
		// Echo text messages for now (implement chat logic here)
		zap.L().Info("Received text message",
			zap.String("user_id", c.UserID),
			zap.Any("data", message.Data))

	default:
		zap.L().Warn("Unknown message type",
			zap.String("type", message.Type),
			zap.String("user_id", c.UserID))
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}