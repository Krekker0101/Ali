package agent

import (
	"context"
	"time"
)

// AgentType defines the type of AI agent
type AgentType string

const (
	// LocalAgent uses local LLM models (Ollama)
	LocalAgent AgentType = "local"
	// CloudAgent uses cloud-based AI services (OpenAI, Claude, etc.)
	CloudAgent AgentType = "cloud"
	// HybridAgent can use both local and cloud models
	HybridAgent AgentType = "hybrid"
)

// AgentCapability defines what an agent can do
type AgentCapability string

const (
	// CodeAnalysis capability to analyze code
	CodeAnalysis AgentCapability = "code_analysis"
	// CodeGeneration capability to generate code
	CodeGeneration AgentCapability = "code_generation"
	// CodeRefactoring capability to refactor code
	CodeRefactoring AgentCapability = "code_refactoring"
	// BugFix capability to identify and fix bugs
	BugFix AgentCapability = "bug_fix"
	// Documentation capability to generate documentation
	Documentation AgentCapability = "documentation"
	// Testing capability to generate tests
	Testing AgentCapability = "testing"
	// ProjectAnalysis capability to analyze projects
	ProjectAnalysis AgentCapability = "project_analysis"
	// PerformanceOptimization capability to optimize code
	PerformanceOptimization AgentCapability = "performance_optimization"
	// SecurityAudit capability to audit code for security
	SecurityAudit AgentCapability = "security_audit"
)

// Agent represents an AI agent
type Agent struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Type         AgentType             `json:"type"`
	Model        string                `json:"model"`
	Capabilities []AgentCapability     `json:"capabilities"`
	Config       AgentConfig           `json:"config"`
	Status       string                `json:"status"` // "active", "inactive", "error"
	CreatedAt    time.Time             `json:"created_at"`
	LastUsed     *time.Time            `json:"last_used,omitempty"`
	Stats        AgentStats            `json:"stats"`
}

// AgentConfig contains agent configuration
type AgentConfig struct {
	Temperature      float32          `json:"temperature"`
	TopP             float32          `json:"top_p"`
	TopK             int              `json:"top_k"`
	MaxTokens        int              `json:"max_tokens"`
	Timeout          time.Duration    `json:"timeout"`
	RetryAttempts    int              `json:"retry_attempts"`
	RetryDelay       time.Duration    `json:"retry_delay"`
	SystemPrompt     string           `json:"system_prompt,omitempty"`
	CustomParameters map[string]any   `json:"custom_parameters,omitempty"`
}

// AgentStats contains agent statistics
type AgentStats struct {
	TotalRequests   int64         `json:"total_requests"`
	SuccessCount    int64         `json:"success_count"`
	ErrorCount      int64         `json:"error_count"`
	TotalTokens     int64         `json:"total_tokens"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastError       string        `json:"last_error,omitempty"`
}

// TaskRequest represents a task request to an agent
type TaskRequest struct {
	ID          string                 `json:"id"`
	AgentID     string                 `json:"agent_id"`
	Type        string                 `json:"type"`
	Context     string                 `json:"context"`
	Prompt      string                 `json:"prompt"`
	Files       []TaskFile             `json:"files,omitempty"`
	Parameters  map[string]any         `json:"parameters,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
	CreatedAt   time.Time              `json:"created_at"`
}

// TaskFile represents a file associated with a task
type TaskFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Type    string `json:"type"` // "source", "config", "test", etc.
}

// TaskResponse represents a response from an agent
type TaskResponse struct {
	ID         string               `json:"id"`
	RequestID  string               `json:"request_id"`
	AgentID    string               `json:"agent_id"`
	Status     string               `json:"status"` // "pending", "processing", "completed", "failed"
	Result     string               `json:"result"`
	Output     []AgentOutput        `json:"output"`
	Suggestions []string            `json:"suggestions"`
	Confidence float32              `json:"confidence"` // 0.0 to 1.0
	TokensUsed int                  `json:"tokens_used"`
	Duration   time.Duration        `json:"duration"`
	Error      string               `json:"error,omitempty"`
	CompletedAt *time.Time          `json:"completed_at,omitempty"`
}

// AgentOutput represents output from an agent
type AgentOutput struct {
	Type    string `json:"type"` // "code", "text", "error", "warning"
	Content string `json:"content"`
	File    string `json:"file,omitempty"`
}

// CodeAnalysisTask represents a code analysis task
type CodeAnalysisTask struct {
	Files      []TaskFile      `json:"files"`
	AnalysisType string        `json:"analysis_type"` // "quality", "security", "performance", "all"
	IncludeMetrics bool        `json:"include_metrics"`
}

// CodeGenerationTask represents a code generation task
type CodeGenerationTask struct {
	Specification string        `json:"specification"`
	Language      string        `json:"language"`
	Framework     string        `json:"framework,omitempty"`
	BuildTests    bool          `json:"build_tests"`
	AddDocumentation bool       `json:"add_documentation"`
}

// CodeRefactoringTask represents a code refactoring task
type CodeRefactoringTask struct {
	Files      []TaskFile      `json:"files"`
	Goals      []string        `json:"goals"` // "readability", "performance", "security", etc.
	PreserveAPI bool           `json:"preserve_api"`
	PreserveBehavior bool      `json:"preserve_behavior"`
}

// BugFixTask represents a bug fix task
type BugFixTask struct {
	Files      []TaskFile      `json:"files"`
	BugDescription string      `json:"bug_description"`
	ReproductionSteps []string `json:"reproduction_steps,omitempty"`
}

// ProjectAnalysisTask represents a project analysis task
type ProjectAnalysisTask struct {
	RootPath       string                `json:"root_path"`
	IncludeMetrics bool                  `json:"include_metrics"`
	AnalysisDepth  string                `json:"analysis_depth"` // "shallow", "medium", "deep"
}

// AgentProvider provides agent management
type AgentProvider interface {
	// CreateAgent creates a new agent
	CreateAgent(ctx context.Context, agent *Agent) (*Agent, error)
	
	// GetAgent retrieves an agent by ID
	GetAgent(ctx context.Context, agentID string) (*Agent, error)
	
	// ListAgents lists all agents
	ListAgents(ctx context.Context) ([]*Agent, error)
	
	// UpdateAgent updates an agent
	UpdateAgent(ctx context.Context, agent *Agent) (*Agent, error)
	
	// DeleteAgent deletes an agent
	DeleteAgent(ctx context.Context, agentID string) error
	
	// ExecuteTask executes a task with an agent
	ExecuteTask(ctx context.Context, taskReq *TaskRequest) (*TaskResponse, error)
	
	// GetTaskStatus gets the status of a task
	GetTaskStatus(ctx context.Context, taskID string) (*TaskResponse, error)
	
	// CancelTask cancels a task
	CancelTask(ctx context.Context, taskID string) error
}

// LocalAgentProvider provides local agent implementation
type LocalAgentProvider interface {
	AgentProvider
	
	// GetAvailableModels returns available local models
	GetAvailableModels(ctx context.Context) ([]string, error)
}

// CloudAgentProvider provides cloud agent implementation
type CloudAgentProvider interface {
	AgentProvider
	
	// Authenticate authenticates with cloud service
	Authenticate(ctx context.Context, credentials map[string]string) error
	
	// IsAuthenticated checks if authenticated
	IsAuthenticated(ctx context.Context) (bool, error)
}

// DefaultAgentConfig returns default agent configuration
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		Temperature:   0.7,
		TopP:          0.9,
		TopK:          40,
		MaxTokens:     2048,
		Timeout:       5 * time.Minute,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
		SystemPrompt:  "You are a helpful code assistant. Provide clear, accurate responses.",
	}
}

// NewAgent creates a new agent
func NewAgent(id, name string, agentType AgentType, model string, capabilities []AgentCapability) *Agent {
	return &Agent{
		ID:           id,
		Name:         name,
		Type:         agentType,
		Model:        model,
		Capabilities: capabilities,
		Config:       DefaultAgentConfig(),
		Status:       "inactive",
		CreatedAt:    time.Now(),
		Stats: AgentStats{
			TotalRequests: 0,
			SuccessCount:  0,
			ErrorCount:    0,
		},
	}
}

// HasCapability checks if agent has a capability
func (a *Agent) HasCapability(capability AgentCapability) bool {
	for _, cap := range a.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// IsActive checks if agent is active
func (a *Agent) IsActive() bool {
	return a.Status == "active"
}

// UpdateStats updates agent statistics
func (a *Agent) UpdateStats(success bool, tokensUsed int, duration time.Duration) {
	a.Stats.TotalRequests++
	a.Stats.TotalTokens += int64(tokensUsed)
	
	if success {
		a.Stats.SuccessCount++
	} else {
		a.Stats.ErrorCount++
	}
	
	now := time.Now()
	a.LastUsed = &now
}
