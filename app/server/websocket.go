package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/agent"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
}

// AgentRequest represents an AI agent request from the client
type AgentRequest struct {
	Provider string          `json:"provider"`
	Model    string          `json:"model"`
	Prompt   string          `json:"prompt"`
	Context  json.RawMessage `json:"context,omitempty"`
}

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan WebSocketMessage
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
	agentPool  *agent.LocalAgentImpl
}

// WebSocketClient represents a connected client
type WebSocketClient struct {
	hub      *WebSocketHub
	conn     *websocket.Conn
	send     chan WebSocketMessage
	clientID string
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(agentPool *agent.LocalAgentImpl) *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan WebSocketMessage, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		agentPool:  agentPool,
	}
}

// Start runs the hub
func (h *WebSocketHub) Start() {
	go func() {
		for {
			select {
			case client := <-h.register:
				h.mu.Lock()
				h.clients[client] = true
				h.mu.Unlock()
				log.Printf("WebSocket client connected: %s", client.clientID)

			case client := <-h.unregister:
				h.mu.Lock()
				if _, ok := h.clients[client]; ok {
					delete(h.clients, client)
					close(client.send)
				}
				h.mu.Unlock()
				log.Printf("WebSocket client disconnected: %s", client.clientID)

			case message := <-h.broadcast:
				h.mu.RLock()
				for client := range h.clients {
					select {
					case client.send <- message:
					default:
						// Client's send channel is full, drop message
						log.Printf("Client %s send channel full, dropping message", client.clientID)
					}
				}
				h.mu.RUnlock()
			}
		}
	}()
}

// HandleAgentWebSocket handles WebSocket connections for AI agent
func (h *WebSocketHub) HandleAgentWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &WebSocketClient{
		hub:      h,
		conn:     conn,
		send:     make(chan WebSocketMessage, 256),
		clientID: fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	h.register <- client

	go client.readLoop()
	go client.writeLoop()
}

// readLoop reads messages from the WebSocket connection
func (c *WebSocketClient) readLoop() {
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
		var msg WebSocketMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		// Handle different message types
		switch msg.Type {
		case "agent-request":
			go c.handleAgentRequest(&msg)
		case "ping":
			c.send <- WebSocketMessage{
				Type:      "pong",
				Timestamp: time.Now().UnixNano() / 1e6,
			}
		}
	}
}

// writeLoop writes messages to the WebSocket connection
func (c *WebSocketClient) writeLoop() {
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

			if err := c.conn.WriteJSON(message); err != nil {
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

// handleAgentRequest processes agent requests
func (c *WebSocketClient) handleAgentRequest(msg *WebSocketMessage) {
	var req AgentRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		c.sendError("Invalid request")
		return
	}

	if strings.TrimSpace(req.Provider) == "" {
		req.Provider = "local"
	}
	if req.Provider != "local" {
		c.sendError("WebSocket agent currently supports local models only")
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		c.sendError("Model is required")
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		c.sendError("Prompt is required")
		return
	}

	c.sendPayload("task-update", map[string]any{
		"status":   "running",
		"progress": 5,
		"message":  "Preparing local agent...",
	})

	// Execute agent request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if c.hub.agentPool == nil {
		c.sendError("Agent runtime is not available")
		return
	}

	localAgent := agent.NewAgent("", "IDE WebSocket Agent", agent.LocalAgent, req.Model, []agent.AgentCapability{
		agent.CodeAnalysis,
		agent.CodeGeneration,
		agent.CodeRefactoring,
		agent.BugFix,
		agent.Documentation,
		agent.Testing,
	})
	createdAgent, err := c.hub.agentPool.CreateAgent(ctx, localAgent)
	if err != nil {
		c.sendError(err.Error())
		return
	}

	task, err := c.hub.agentPool.ExecuteTask(ctx, &agent.TaskRequest{
		AgentID:   createdAgent.ID,
		Type:      "generic",
		Context:   string(req.Context),
		Prompt:    req.Prompt,
		Timeout:   5 * time.Minute,
		CreatedAt: time.Now(),
	})
	if err != nil {
		c.sendError(err.Error())
		return
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.sendError(ctx.Err().Error())
			return
		case <-ticker.C:
			status, err := c.hub.agentPool.GetTaskStatus(ctx, task.ID)
			if err != nil {
				c.sendError(err.Error())
				return
			}
			c.sendPayload("task-update", map[string]any{
				"id":       status.ID,
				"status":   status.Status,
				"progress": agentTaskProgress(status.Status),
				"message":  "Processing...",
			})
			switch status.Status {
			case "completed":
				c.sendPayload("chat", map[string]any{
					"content": status.Result,
					"role":    "assistant",
				})
				c.sendPayload("task-complete", map[string]any{
					"id":       status.ID,
					"status":   "completed",
					"progress": 100,
				})
				return
			case "failed":
				if status.Error == "" {
					status.Error = "Agent task failed"
				}
				c.sendError(status.Error)
				return
			}
		}
	}
}

func agentTaskProgress(status string) int {
	switch status {
	case "pending":
		return 15
	case "processing":
		return 45
	case "completed":
		return 100
	case "failed":
		return 100
	default:
		return 20
	}
}

func (c *WebSocketClient) sendError(message string) {
	c.sendPayload("error", map[string]string{"error": message})
}

func (c *WebSocketClient) sendPayload(msgType string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte(`{"error":"failed to encode websocket payload"}`)
		msgType = "error"
	}
	c.send <- WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().UnixNano() / 1e6,
	}
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) Broadcast(msgType string, data interface{}) {
	payload, _ := json.Marshal(data)
	h.broadcast <- WebSocketMessage{
		Type:      msgType,
		Data:      payload,
		Timestamp: time.Now().UnixNano() / 1e6,
	}
}
