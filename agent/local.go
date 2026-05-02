package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
)

// LocalAgentImpl implements LocalAgentProvider using Ollama
type LocalAgentImpl struct {
	client     *api.Client
	agents     map[string]*Agent
	tasks      map[string]*TaskResponse
	mu         sync.RWMutex
	pullMu     sync.Mutex
	taskSem    chan struct{} // Semaphore for concurrent tasks
	maxWorkers int
}

// NewLocalAgent creates a new local agent provider
func NewLocalAgent(client *api.Client, maxWorkers int) *LocalAgentImpl {
	return &LocalAgentImpl{
		client:     client,
		agents:     make(map[string]*Agent),
		tasks:      make(map[string]*TaskResponse),
		taskSem:    make(chan struct{}, maxWorkers),
		maxWorkers: maxWorkers,
	}
}

// CreateAgent creates a new agent
func (p *LocalAgentImpl) CreateAgent(ctx context.Context, agent *Agent) (*Agent, error) {
	if agent.ID == "" {
		agent.ID = "agent_" + uuid.New().String()
	}

	agent.CreatedAt = time.Now()
	agent.Status = "active"

	p.mu.Lock()
	defer p.mu.Unlock()

	p.agents[agent.ID] = agent
	return agent, nil
}

// GetAgent retrieves an agent by ID
func (p *LocalAgentImpl) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	agent, ok := p.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return agent, nil
}

// ListAgents lists all agents
func (p *LocalAgentImpl) ListAgents(ctx context.Context) ([]*Agent, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	agents := make([]*Agent, 0, len(p.agents))
	for _, agent := range p.agents {
		agents = append(agents, agent)
	}

	return agents, nil
}

// UpdateAgent updates an agent
func (p *LocalAgentImpl) UpdateAgent(ctx context.Context, agent *Agent) (*Agent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.agents[agent.ID]; !ok {
		return nil, fmt.Errorf("agent not found: %s", agent.ID)
	}

	p.agents[agent.ID] = agent
	return agent, nil
}

// DeleteAgent deletes an agent
func (p *LocalAgentImpl) DeleteAgent(ctx context.Context, agentID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.agents[agentID]; !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	delete(p.agents, agentID)
	return nil
}

// ExecuteTask executes a task with an agent
func (p *LocalAgentImpl) ExecuteTask(ctx context.Context, taskReq *TaskRequest) (*TaskResponse, error) {
	// Validate agent exists
	agent, err := p.GetAgent(ctx, taskReq.AgentID)
	if err != nil {
		return nil, err
	}

	if !agent.IsActive() {
		return nil, fmt.Errorf("agent is not active: %s", taskReq.AgentID)
	}

	// Create task response
	if taskReq.ID == "" {
		taskReq.ID = "task_" + uuid.New().String()
	}

	taskResp := &TaskResponse{
		ID:        taskReq.ID,
		RequestID: taskReq.ID,
		AgentID:   taskReq.AgentID,
		Status:    "pending",
	}

	p.mu.Lock()
	p.tasks[taskReq.ID] = taskResp
	p.mu.Unlock()

	// Execute task asynchronously with semaphore
	go func() {
		startTime := time.Now()
		defer func() {
			taskResp.Duration = time.Since(startTime)
			agent.UpdateStats(taskResp.Error == "", len(taskResp.Result), taskResp.Duration)
		}()

		// Acquire semaphore
		select {
		case p.taskSem <- struct{}{}:
		case <-ctx.Done():
			taskResp.Status = "failed"
			taskResp.Error = "context cancelled"
			return
		}
		defer func() { <-p.taskSem }()

		// Update status
		p.mu.Lock()
		taskResp.Status = "processing"
		p.mu.Unlock()

		// Execute based on task type
		switch taskReq.Type {
		case "code_analysis":
			p.executeCodeAnalysis(ctx, agent, taskReq, taskResp)
		case "code_generation":
			p.executeCodeGeneration(ctx, agent, taskReq, taskResp)
		case "bug_fix":
			p.executeBugFix(ctx, agent, taskReq, taskResp)
		case "refactoring":
			p.executeRefactoring(ctx, agent, taskReq, taskResp)
		default:
			p.executeGenericTask(ctx, agent, taskReq, taskResp)
		}

		if taskResp.Error == "" {
			taskResp.Status = "completed"
		} else {
			taskResp.Status = "failed"
		}

		completedAt := time.Now()
		taskResp.CompletedAt = &completedAt
	}()

	return taskResp, nil
}

// GetTaskStatus gets the status of a task
func (p *LocalAgentImpl) GetTaskStatus(ctx context.Context, taskID string) (*TaskResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	task, ok := p.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// CancelTask cancels a task
func (p *LocalAgentImpl) CancelTask(ctx context.Context, taskID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	task, ok := p.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status == "processing" || task.Status == "pending" {
		task.Status = "cancelled"
		now := time.Now()
		task.CompletedAt = &now
	}

	return nil
}

// GetAvailableModels returns available local models
func (p *LocalAgentImpl) GetAvailableModels(ctx context.Context) ([]string, error) {
	listResp, err := p.client.List(ctx)
	if err != nil {
		return recommendedLocalModels(), nil
	}

	models := make([]string, 0, len(listResp.Models))
	for _, model := range listResp.Models {
		if model.Model != "" {
			models = append(models, model.Model)
			continue
		}
		models = append(models, model.Name)
	}

	return mergeModelNames(models, recommendedLocalModels()), nil
}

// executeCodeAnalysis analyzes code using the agent
func (p *LocalAgentImpl) executeCodeAnalysis(ctx context.Context, agent *Agent, req *TaskRequest, resp *TaskResponse) {
	prompt := fmt.Sprintf(`Analyze the following code and provide:
1. Code quality assessment
2. Potential bugs or issues
3. Performance concerns
4. Security vulnerabilities
5. Suggestions for improvement

Context: %s

Prompt: %s

Code:
%s`, req.Context, req.Prompt, p.getFileContents(req.Files))

	result, err := p.callOllamaAPI(ctx, agent.Model, prompt, agent.Config)
	if err != nil {
		resp.Error = err.Error()
		return
	}

	resp.Result = result
	resp.Output = append(resp.Output, AgentOutput{
		Type:    "text",
		Content: result,
	})
}

// executeCodeGeneration generates code using the agent
func (p *LocalAgentImpl) executeCodeGeneration(ctx context.Context, agent *Agent, req *TaskRequest, resp *TaskResponse) {
	prompt := fmt.Sprintf(`Generate code based on the following specification:

Specification: %s
Context: %s
Prompt: %s

Please provide clean, well-documented, production-ready code.`, req.Prompt, req.Context, req.Prompt)

	result, err := p.callOllamaAPI(ctx, agent.Model, prompt, agent.Config)
	if err != nil {
		resp.Error = err.Error()
		return
	}

	resp.Result = result
	resp.Output = append(resp.Output, AgentOutput{
		Type:    "code",
		Content: result,
	})
}

// executeBugFix fixes bugs using the agent
func (p *LocalAgentImpl) executeBugFix(ctx context.Context, agent *Agent, req *TaskRequest, resp *TaskResponse) {
	prompt := fmt.Sprintf(`Fix the bug in the following code:

Description: %s
Context: %s
Code:
%s

Please provide:
1. Root cause analysis
2. Fixed code
3. Explanation of the fix`, req.Prompt, req.Context, p.getFileContents(req.Files))

	result, err := p.callOllamaAPI(ctx, agent.Model, prompt, agent.Config)
	if err != nil {
		resp.Error = err.Error()
		return
	}

	resp.Result = result
	resp.Output = append(resp.Output, AgentOutput{
		Type:    "code",
		Content: result,
	})
}

// executeRefactoring refactors code using the agent
func (p *LocalAgentImpl) executeRefactoring(ctx context.Context, agent *Agent, req *TaskRequest, resp *TaskResponse) {
	prompt := fmt.Sprintf(`Refactor the following code to improve:
- Readability
- Maintainability
- Performance
- Following best practices

Context: %s
Prompt: %s

Code:
%s`, req.Context, req.Prompt, p.getFileContents(req.Files))

	result, err := p.callOllamaAPI(ctx, agent.Model, prompt, agent.Config)
	if err != nil {
		resp.Error = err.Error()
		return
	}

	resp.Result = result
	resp.Output = append(resp.Output, AgentOutput{
		Type:    "code",
		Content: result,
	})
}

// executeGenericTask executes a generic task
func (p *LocalAgentImpl) executeGenericTask(ctx context.Context, agent *Agent, req *TaskRequest, resp *TaskResponse) {
	prompt := fmt.Sprintf(`Context: %s

Request: %s

Files:
%s`, req.Context, req.Prompt, p.getFileContents(req.Files))

	result, err := p.callOllamaAPI(ctx, agent.Model, prompt, agent.Config)
	if err != nil {
		resp.Error = err.Error()
		return
	}

	resp.Result = result
	resp.Output = append(resp.Output, AgentOutput{
		Type:    "text",
		Content: result,
	})
}

// callOllamaAPI calls the Ollama API
func (p *LocalAgentImpl) callOllamaAPI(ctx context.Context, model, prompt string, config AgentConfig) (string, error) {
	if err := p.ensureModel(ctx, model); err != nil {
		return "", err
	}

	// Create GenerateRequest
	req := &api.GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Options: map[string]any{
			"temperature": config.Temperature,
			"top_p":       config.TopP,
			"top_k":       config.TopK,
			"num_predict": config.MaxTokens,
		},
		Stream: new(bool), // Set to false for non-streaming
	}

	// Set stream to false
	*req.Stream = false

	var fullResponse strings.Builder

	// Call Ollama API
	respFunc := func(resp api.GenerateResponse) error {
		fullResponse.WriteString(resp.Response)
		return nil
	}

	err := p.client.Generate(ctx, req, respFunc)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w", err)
	}

	result := fullResponse.String()
	if result == "" {
		return "", fmt.Errorf("empty response from Ollama API")
	}

	return result, nil
}

func (p *LocalAgentImpl) ensureModel(ctx context.Context, model string) error {
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required")
	}
	if ok := p.hasModel(ctx, model); ok {
		return nil
	}

	p.pullMu.Lock()
	defer p.pullMu.Unlock()

	if ok := p.hasModel(ctx, model); ok {
		return nil
	}

	err := p.client.Pull(ctx, &api.PullRequest{Model: model}, func(api.ProgressResponse) error {
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to download local model %q: %w", model, err)
	}
	return nil
}

func (p *LocalAgentImpl) hasModel(ctx context.Context, model string) bool {
	listResp, err := p.client.List(ctx)
	if err != nil {
		return false
	}
	for _, item := range listResp.Models {
		if sameModelName(item.Model, model) || sameModelName(item.Name, model) {
			return true
		}
	}
	return false
}

func recommendedLocalModels() []string {
	return []string{
		"qwen2.5-coder:1.5b",
		"qwen2.5-coder:7b",
		"llama3.2:3b",
		"mistral:7b",
		"codellama:7b",
	}
}

func mergeModelNames(installed, recommended []string) []string {
	merged := make([]string, 0, len(installed)+len(recommended))
	for _, name := range append(installed, recommended...) {
		seen := false
		for _, existing := range merged {
			if sameModelName(existing, name) {
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

func sameModelName(a, b string) bool {
	normalize := func(s string) string {
		s = strings.TrimSpace(strings.ToLower(s))
		s = strings.TrimSuffix(s, ":latest")
		return s
	}
	return normalize(a) == normalize(b)
}

// getFileContents returns formatted file contents
func (p *LocalAgentImpl) getFileContents(files []TaskFile) string {
	var contents string
	for _, file := range files {
		contents += fmt.Sprintf("File: %s\n%s\n---\n", file.Path, file.Content)
	}
	return contents
}
