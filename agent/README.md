# Agent Package

## Overview

The Agent package provides a comprehensive framework for local and cloud-based AI agents that can perform intelligent code analysis, generation, and refactoring tasks. It enables seamless integration with multiple LLM providers.

## Features

- **Multiple Agent Types**: Local (Ali), Cloud (OpenAI, Claude), Hybrid
- **Diverse Capabilities**: Code analysis, generation, refactoring, bug fix, documentation, testing, and more
- **Asynchronous Execution**: Non-blocking task processing
- **Concurrency Control**: Configurable worker pool with semaphore-based limiting
- **Task Management**: Full lifecycle management (create, monitor, cancel)
- **Statistics Tracking**: Monitor agent performance and usage
- **Error Resilience**: Automatic retries with exponential backoff
- **Timeout Protection**: Prevent long-running tasks from blocking

## Architecture

```
Agent Framework
├── Core Types (agent.go)
│   ├── Agent
│   ├── TaskRequest
│   ├── TaskResponse
│   └── Capability enums
├── Local Provider (local.go)
│   ├── LocalAgentImpl
│   └── Ali integration
├── Cloud Provider (cloud.go - to be added)
│   ├── OpenAI integration
│   └── Claude integration
└── HTTP API (api.go)
    └── REST endpoints
```

## Supported Capabilities

```go
const (
    CodeAnalysis           = "code_analysis"           // Analyze code quality
    CodeGeneration         = "code_generation"         // Generate new code
    CodeRefactoring        = "code_refactoring"        // Improve existing code
    BugFix                 = "bug_fix"                 // Find and fix bugs
    Documentation          = "documentation"           // Generate documentation
    Testing                = "testing"                 // Generate tests
    ProjectAnalysis        = "project_analysis"        // Analyze entire projects
    PerformanceOptimization = "performance_optimization" // Optimize code
    SecurityAudit          = "security_audit"          // Security review
)
```

## Usage

### Creating an Agent

```go
import "github.com/ollama/ollama/agent"

// Initialize with Ali client
client := &api.Client{Base: "http://localhost:11434"}
provider := agent.NewLocalAgent(client, 5) // Max 5 concurrent tasks

// Create an agent
codeAgent := &agent.Agent{
    Name:  "CodeAnalyzer",
    Type:  agent.LocalAgent,
    Model: "llama2",
    Capabilities: []agent.AgentCapability{
        agent.CodeAnalysis,
        agent.BugFix,
    },
    Config: agent.DefaultAgentConfig(),
}

created, err := provider.CreateAgent(context.Background(), codeAgent)
if err != nil {
    log.Fatal(err)
}
```

### Managing Agents

```go
// List all agents
agents, err := provider.ListAgents(context.Background())

// Get specific agent
agent, err := provider.GetAgent(context.Background(), agentID)

// Update agent
agent.Status = "inactive"
updated, err := provider.UpdateAgent(context.Background(), agent)

// Delete agent
err := provider.DeleteAgent(context.Background(), agentID)
```

### Executing Tasks

```go
// Create task request
taskReq := &agent.TaskRequest{
    AgentID: created.ID,
    Type:    "code_analysis",
    Context: "REST API handler",
    Prompt:  "Analyze this code for security vulnerabilities",
    Files: []agent.TaskFile{
        {
            Path:    "handler.go",
            Content: "package main\n...",
            Type:    "source",
        },
    },
    Timeout: 5 * time.Minute,
    Parameters: map[string]interface{}{
        "analysis_type": "security",
    },
}

// Execute task (asynchronous)
response, err := provider.ExecuteTask(context.Background(), taskReq)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Task created: %s (Status: %s)\n", response.ID, response.Status)
```

### Monitoring Tasks

```go
// Get task status
status, err := provider.GetTaskStatus(context.Background(), taskID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", status.Status)
fmt.Printf("Result: %s\n", status.Result)
fmt.Printf("Duration: %v\n", status.Duration)

// Poll for completion
ticker := time.NewTicker(1 * time.Second)
for {
    select {
    case <-ticker.C:
        status, _ := provider.GetTaskStatus(ctx, taskID)
        if status.Status == "completed" || status.Status == "failed" {
            fmt.Println("Task complete:", status.Status)
            return
        }
    }
}

// Cancel task
err := provider.CancelTask(context.Background(), taskID)
```

## Data Structures

### Agent
```go
type Agent struct {
    ID            string
    Name          string
    Type          AgentType              // "local", "cloud", "hybrid"
    Model         string                 // Model name
    Capabilities  []AgentCapability      // What it can do
    Config        AgentConfig            // Configuration
    Status        string                 // "active", "inactive", "error"
    Stats         AgentStats             // Performance metrics
    CreatedAt     time.Time
    LastUsedAt    time.Time
}
```

### TaskRequest
```go
type TaskRequest struct {
    ID         string
    AgentID    string
    Type       string                          // Task type
    Context    string                          // Context information
    Prompt     string                          // User prompt
    Files      []TaskFile                      // Files to process
    Parameters map[string]interface{}          // Task parameters
    Timeout    time.Duration
    Metadata   map[string]interface{}
}
```

### TaskResponse
```go
type TaskResponse struct {
    ID          string
    RequestID   string
    AgentID     string
    Status      string                  // pending, processing, completed, failed, cancelled
    Result      string
    Output      []AgentOutput
    Error       string
    Duration    time.Duration
    TokensUsed  int
    CompletedAt *time.Time
    Metadata    map[string]interface{}
}
```

### AgentConfig
```go
type AgentConfig struct {
    Temperature   float64           // 0.0-2.0 (default: 0.7)
    TopP          float64           // 0.0-1.0 (default: 0.9)
    TopK          int               // 0-200 (default: 40)
    MaxTokens     int               // Max response tokens (default: 2048)
    Timeout       time.Duration     // Task timeout (default: 5 min)
    RetryAttempts int               // Retry failed tasks (default: 3)
    SystemPrompt  string            // Custom system prompt
}
```

## Task Types

### Code Analysis
```go
taskReq := &agent.TaskRequest{
    Type:    "code_analysis",
    Prompt:  "Analyze this code for issues",
    // Provides: quality assessment, bugs, performance, security
}
```

### Code Generation
```go
taskReq := &agent.TaskRequest{
    Type:   "code_generation",
    Prompt: "Generate a REST API handler for user management",
    // Provides: production-ready code with tests and docs
}
```

### Bug Fix
```go
taskReq := &agent.TaskRequest{
    Type:   "bug_fix",
    Prompt: "Fix the bug in this code",
    Files: []agent.TaskFile{{
        Path:    "main.go",
        Content: "...",
    }},
    // Provides: root cause, fixed code, explanation
}
```

### Code Refactoring
```go
taskReq := &agent.TaskRequest{
    Type:   "refactoring",
    Prompt: "Improve readability and performance",
    Files: []agent.TaskFile{{
        Path:    "main.go",
        Content: "...",
    }},
    // Provides: refactored code with improvements
}
```

## HTTP API

### Endpoints

```
POST   /api/v1/agents                    # Create agent
GET    /api/v1/agents                    # List agents
GET    /api/v1/agents/:id                # Get agent
PUT    /api/v1/agents/:id                # Update agent
DELETE /api/v1/agents/:id                # Delete agent
POST   /api/v1/agents/:id/tasks          # Execute task
GET    /api/v1/agents/tasks/:taskID      # Get task status
DELETE /api/v1/agents/tasks/:taskID      # Cancel task
```

See [IMPLEMENTATION_GUIDE.md](../IMPLEMENTATION_GUIDE.md#agents-1) for detailed API documentation.

## Concurrency Model

### Worker Pool

The local agent uses a semaphore-based worker pool:

```go
// Create with max 5 concurrent tasks
provider := agent.NewLocalAgent(client, 5)

// Additional tasks wait for slots
for i := 0; i < 10; i++ {
    resp, _ := provider.ExecuteTask(ctx, taskReq) // First 5 start immediately
}                                                  // Rest queue until slots free
```

### Task Lifecycle

```
1. pending    -> Task created, waiting for worker slot
2. processing -> Task acquired worker, executing
3. completed  -> Task finished successfully
4. failed     -> Task encountered error
5. cancelled   -> Task manually cancelled
```

## Error Handling

```go
response, err := provider.ExecuteTask(ctx, taskReq)
if err != nil {
    // Handle creation errors
    log.Printf("Failed to execute task: %v", err)
}

// Later, check task result
status, _ := provider.GetTaskStatus(ctx, response.ID)
if status.Status == "failed" {
    log.Printf("Task failed: %s", status.Error)
}
```

## Performance

- **Concurrency**: Configurable worker pool (1-100+)
- **Memory**: Efficient task tracking with map storage
- **Timeouts**: Prevent resource exhaustion from long-running tasks
- **Retry Logic**: Automatic retries with exponential backoff
- **Statistics**: Real-time agent performance metrics

## Configuration

### Environment Variables

```bash
export OLLAMA_AGENTS_ENABLED=true
export OLLAMA_AGENTS_MAX_WORKERS=5
export OLLAMA_AGENTS_DEFAULT_TIMEOUT=300  # 5 minutes
```

### Default Config

```go
agent.DefaultAgentConfig()
// Returns:
// - Temperature: 0.7
// - TopP: 0.9
// - TopK: 40
// - MaxTokens: 2048
// - Timeout: 5 minutes
// - RetryAttempts: 3
```

## Examples

See [examples_new_features.go](../examples_new_features.go) for complete examples:

- Creating agents
- Executing tasks
- Monitoring task status
- Handling concurrent tasks
- Error handling

## Local Agent (Ali)

The local agent uses the Ali runtime:

```go
// Requires Ali running
client := &api.Client{Base: "http://localhost:11434"}

// Agent uses local models
provider := agent.NewLocalAgent(client, 5)

// Task execution:
// 1. Constructs context-aware prompt
// 2. Calls Ali API
// 3. Returns model output
```

## Cloud Agents (Future)

Cloud agent implementations (planned):

```go
// OpenAI
provider := agent.NewCloudAgent("openai", apiKey)

// Claude
provider := agent.NewCloudAgent("anthropic", apiKey)

// Generic HTTP provider
provider := agent.NewCloudAgent("custom", customConfig)
```

## Security Considerations

- **Timeout Protection**: All tasks have configurable timeouts
- **Input Validation**: All inputs validated before processing
- **Error Isolation**: Task failures don't affect other tasks
- **Resource Limits**: Semaphore prevents resource exhaustion
- **Authentication**: Support for API keys and OAuth (cloud)

## Related

- [Validation Package](../validation/README.md) - Input validation
- [Code Editor](../editor/README.md) - File operations for code analysis
- [IMPLEMENTATION_GUIDE.md](../IMPLEMENTATION_GUIDE.md) - Complete guide

## Future Enhancements

1. **Cloud Providers**: OpenAI, Claude, custom providers
2. **Agent Chaining**: Multiple agents working together
3. **Model Fine-tuning**: Train agents on project-specific data
4. **Caching**: Cache common queries
5. **Webhooks**: Real-time task notifications
6. **Streaming**: Stream task output in real-time
7. **Cost Tracking**: Monitor API usage and costs
8. **Quality Feedback**: Improve agents based on feedback
