package agent

import (
	"context"
	"net/http"
	
	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/validation"
)

// AgentAPI provides HTTP handlers for agent operations
type AgentAPI struct {
	provider AgentProvider
}

// NewAgentAPI creates a new agent API
func NewAgentAPI(provider AgentProvider) *AgentAPI {
	return &AgentAPI{
		provider: provider,
	}
}

// CreateAgent creates a new agent
func (api *AgentAPI) CreateAgent(c *gin.Context) {
	var agent Agent
	
	if err := c.ShouldBindJSON(&agent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate
	val := validation.NewValidator()
	val.ValidateModelName(agent.Name)
	if val.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": val.FirstError().Error()})
		return
	}
	
	created, err := api.provider.CreateAgent(context.Background(), &agent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, created)
}

// GetAgent gets an agent by ID
func (api *AgentAPI) GetAgent(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent ID is required"})
		return
	}
	
	agent, err := api.provider.GetAgent(context.Background(), agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, agent)
}

// ListAgents lists all agents
func (api *AgentAPI) ListAgents(c *gin.Context) {
	agents, err := api.provider.ListAgents(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, agents)
}

// UpdateAgent updates an agent
func (api *AgentAPI) UpdateAgent(c *gin.Context) {
	agentID := c.Param("id")
	var agent Agent
	
	if err := c.ShouldBindJSON(&agent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	agent.ID = agentID
	
	updated, err := api.provider.UpdateAgent(context.Background(), &agent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, updated)
}

// DeleteAgent deletes an agent
func (api *AgentAPI) DeleteAgent(c *gin.Context) {
	agentID := c.Param("id")
	
	if err := api.provider.DeleteAgent(context.Background(), agentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "agent deleted"})
}

// ExecuteTask executes a task
func (api *AgentAPI) ExecuteTask(c *gin.Context) {
	var taskReq TaskRequest
	
	if err := c.ShouldBindJSON(&taskReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate
	val := validation.NewValidator()
	val.ValidateTimeout(int(taskReq.Timeout.Seconds()))
	if val.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": val.FirstError().Error()})
		return
	}
	
	response, err := api.provider.ExecuteTask(context.Background(), &taskReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusAccepted, response)
}

// GetTaskStatus gets task status
func (api *AgentAPI) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("taskID")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task ID is required"})
		return
	}
	
	response, err := api.provider.GetTaskStatus(context.Background(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, response)
}

// CancelTask cancels a task
func (api *AgentAPI) CancelTask(c *gin.Context) {
	taskID := c.Param("taskID")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task ID is required"})
		return
	}
	
	if err := api.provider.CancelTask(context.Background(), taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "task cancelled"})
}

// RegisterAgentRoutes registers agent API routes
func RegisterAgentRoutes(router *gin.Engine, provider AgentProvider) {
	api := NewAgentAPI(provider)
	
	agents := router.Group("/api/v1/agents")
	{
		// Agent management
		agents.POST("", api.CreateAgent)
		agents.GET("", api.ListAgents)
		agents.GET("/:id", api.GetAgent)
		agents.PUT("/:id", api.UpdateAgent)
		agents.DELETE("/:id", api.DeleteAgent)
		
		// Task execution
		agents.POST("/:id/tasks", api.ExecuteTask)
		agents.GET("/tasks/:taskID", api.GetTaskStatus)
		agents.DELETE("/tasks/:taskID", api.CancelTask)
	}
}
