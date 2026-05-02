package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ollama/ollama/agent"
	"github.com/ollama/ollama/api"
)

// ExampleRealOllamaIntegration demonstrates real Ollama API integration
func ExampleRealOllamaIntegration() {
	fmt.Println("=== Real Ollama API Integration Example ===\n")

	// Create Ollama client
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Printf("Failed to create Ollama client: %v\n", err)
		log.Println("Make sure Ollama is running and accessible")
		return
	}

	// Test connection by listing models
	fmt.Println("Testing Ollama connection...")
	models, err := client.List(context.Background())
	if err != nil {
		log.Printf("Failed to list models: %v\n", err)
		log.Println("Make sure Ollama is running and you have downloaded models")
		return
	}

	if len(models.Models) == 0 {
		log.Println("No models found. Please download a model first:")
		log.Println("  ollama pull llama2")
		log.Println("  ollama pull gemma2")
		return
	}

	fmt.Printf("✓ Found %d models:\n", len(models.Models))
	for _, model := range models.Models {
		fmt.Printf("  - %s (%s)\n", model.Name, formatSize(model.Size))
	}
	fmt.Println()

	// Create local agent provider
	provider := agent.NewLocalAgent(client, 3) // Max 3 concurrent tasks

	// Create an agent
	fmt.Println("Creating code analysis agent...")
	codeAgent := &agent.Agent{
		Name:  "CodeAnalyzer",
		Type:  agent.LocalAgent,
		Model: models.Models[0].Name, // Use first available model
		Capabilities: []agent.AgentCapability{
			agent.CodeAnalysis,
			agent.BugFix,
		},
		Config: agent.DefaultAgentConfig(),
	}

	created, err := provider.CreateAgent(context.Background(), codeAgent)
	if err != nil {
		log.Printf("Error creating agent: %v\n", err)
		return
	}

	fmt.Printf("✓ Agent created: %s (ID: %s, Model: %s)\n\n", created.Name, created.ID, created.Model)

	// Execute a real code analysis task
	fmt.Println("Executing real code analysis task...")
	taskReq := &agent.TaskRequest{
		AgentID: created.ID,
		Type:    "code_analysis",
		Context: "Go function analysis",
		Prompt:  "Analyze this Go function for potential issues and improvements",
		Files: []agent.TaskFile{
			{
				Path:    "example.go",
				Content: `package main

import "fmt"

// CalculateSum calculates the sum of numbers
func CalculateSum(numbers []int) int {
	sum := 0
	for i := 0; i < len(numbers); i++ {
		sum += numbers[i]
	}
	return sum
}

func main() {
	numbers := []int{1, 2, 3, 4, 5}
	result := CalculateSum(numbers)
	fmt.Printf("Sum: %d\n", result)
}`,
				Type: "source",
			},
		},
		Timeout: 30 * time.Second, // Shorter timeout for demo
	}

	fmt.Println("Submitting task...")
	taskResp, err := provider.ExecuteTask(context.Background(), taskReq)
	if err != nil {
		log.Printf("Error executing task: %v\n", err)
		return
	}

	fmt.Printf("✓ Task submitted: %s (Status: %s)\n", taskResp.ID, taskResp.Status)

	// Wait for completion with timeout
	fmt.Println("Waiting for completion...")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("❌ Task timed out")
			return
		case <-ticker.C:
			status, err := provider.GetTaskStatus(context.Background(), taskResp.ID)
			if err != nil {
				log.Printf("Error getting task status: %v\n", err)
				return
			}

			fmt.Printf("  Status: %s", status.Status)

			if status.Status == "completed" {
				fmt.Printf(" ✓\n\n")
				fmt.Println("=== ANALYSIS RESULT ===")
				fmt.Println(status.Result)
				fmt.Printf("\n=== METADATA ===\n")
				fmt.Printf("Duration: %v\n", status.Duration)
				fmt.Printf("Tokens used: %d\n", status.TokensUsed)
				return
			} else if status.Status == "failed" {
				fmt.Printf(" ❌\n")
				fmt.Printf("Error: %s\n", status.Error)
				return
			} else if status.Status == "processing" {
				fmt.Printf(" (processing...)\n")
			} else {
				fmt.Printf("\n")
			}
		}
	}
}

// formatSize formats file size in human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func main() {
	fmt.Println("Testing Real Ollama Integration")
	fmt.Println("===============================\n")

	ExampleRealOllamaIntegration()

	fmt.Println("\n===============================")
	fmt.Println("Integration test completed")
}