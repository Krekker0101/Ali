package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ollama/ollama/agent"
)

// IDEServer handles IDE-specific routes
type IDEServer struct {
	wsHub *WebSocketHub
}

// NewIDEServer creates a new IDE server
func NewIDEServer(agentPool *agent.LocalAgentImpl) *IDEServer {
	hub := NewWebSocketHub(agentPool)
	hub.Start()

	return &IDEServer{
		wsHub: hub,
	}
}

// RegisterRoutes registers IDE routes
func (s *IDEServer) RegisterRoutes(mux *http.ServeMux) {
	// WebSocket endpoints
	mux.HandleFunc("/ws/agent", s.wsHub.HandleAgentWebSocket)
	mux.HandleFunc("/ws/tasks", s.wsHub.HandleAgentWebSocket)

	// REST API endpoints
	mux.HandleFunc("/api/v1/editor/files/get", s.handleGetFile)
	mux.HandleFunc("/api/v1/editor/files/create", s.handleCreateFile)
	mux.HandleFunc("/api/v1/editor/files/update", s.handleUpdateFile)
	mux.HandleFunc("/api/v1/editor/files/delete", s.handleDeleteFile)
	mux.HandleFunc("/api/v1/editor/files/list", s.handleListFiles)
	mux.HandleFunc("/api/v1/editor/search", s.handleSearchFiles)

	// Agent endpoints
	mux.HandleFunc("/api/v1/agents/task", s.handleAgentTask)
	mux.HandleFunc("/api/v1/agents/models", s.handleGetModels)
}

// handleGetFile handles GET /api/v1/editor/files/get
func (s *IDEServer) handleGetFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	// TODO: Implement file reading from filesystem
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"path":"` + path + `","content":"// File content here"}`))
}

// handleCreateFile handles POST /api/v1/editor/files/create
func (s *IDEServer) handleCreateFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement file creation
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"success":true}`))
}

// handleUpdateFile handles POST /api/v1/editor/files/update
func (s *IDEServer) handleUpdateFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement file updating
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
}

// handleDeleteFile handles DELETE /api/v1/editor/files/delete
func (s *IDEServer) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement file deletion
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
}

// handleListFiles handles GET /api/v1/editor/files/list
func (s *IDEServer) handleListFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		path = "."
	}

	// TODO: Implement file listing
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"files":[]}`))
}

// handleSearchFiles handles GET /api/v1/editor/search
func (s *IDEServer) handleSearchFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "q parameter required", http.StatusBadRequest)
		return
	}

	// TODO: Implement file searching
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"results":[]}`))
}

// handleAgentTask handles POST /api/v1/agents/task
func (s *IDEServer) handleAgentTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Prompt   string `json:"prompt"`
		Context  any    `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Provider) == "" {
		req.Provider = "local"
	}
	if req.Provider != "local" {
		http.Error(w, "agent task endpoint currently supports local models only", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}
	if s.wsHub == nil || s.wsHub.agentPool == nil {
		http.Error(w, "agent runtime is not available", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	localAgent, err := s.wsHub.agentPool.CreateAgent(ctx, agent.NewAgent("", "IDE REST Agent", agent.LocalAgent, req.Model, []agent.AgentCapability{
		agent.CodeAnalysis,
		agent.CodeGeneration,
		agent.CodeRefactoring,
		agent.BugFix,
		agent.Documentation,
		agent.Testing,
	}))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	contextJSON, _ := json.Marshal(req.Context)
	task, err := s.wsHub.agentPool.ExecuteTask(ctx, &agent.TaskRequest{
		AgentID:   localAgent.ID,
		Type:      "generic",
		Context:   string(contextJSON),
		Prompt:    req.Prompt,
		Timeout:   5 * time.Minute,
		CreatedAt: time.Now(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

// handleGetModels handles GET /api/v1/agents/models
func (s *IDEServer) handleGetModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "local"
	}

	modelNames := []string{
		"qwen2.5-coder:1.5b",
		"qwen2.5-coder:7b",
		"llama3.2:3b",
		"mistral:7b",
		"codellama:7b",
	}
	if provider == "local" && s.wsHub != nil && s.wsHub.agentPool != nil {
		if available, err := s.wsHub.agentPool.GetAvailableModels(r.Context()); err == nil {
			modelNames = mergeServerModelNames(available, modelNames)
		}
	}

	models := make([]map[string]string, 0, len(modelNames))
	for _, name := range modelNames {
		models = append(models, map[string]string{
			"name":        name,
			"description": "Local model",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"models": models})
}

func mergeServerModelNames(installed, recommended []string) []string {
	merged := make([]string, 0, len(installed)+len(recommended))
	for _, name := range append(installed, recommended...) {
		seen := false
		for _, existing := range merged {
			if sameServerModelName(existing, name) {
				seen = true
				break
			}
		}
		if !seen {
			merged = append(merged, name)
		}
	}
	return merged
}

func sameServerModelName(a, b string) bool {
	normalize := func(s string) string {
		s = strings.TrimSpace(strings.ToLower(s))
		s = strings.TrimSuffix(s, ":latest")
		return s
	}
	return normalize(a) == normalize(b)
}
