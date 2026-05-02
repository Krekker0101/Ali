package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ollama/ollama/agent"
	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/editor"
	"github.com/ollama/ollama/validation"
)

// ExampleCodeEditor demonstrates the code editor functionality
func ExampleCodeEditor() {
	fmt.Println("=== Code Editor Example ===\n")

	// Create filesystem manager
	fs := editor.NewLocalFileSystemManager("./projects")

	// Create a directory
	fmt.Println("Creating directory...")
	err := fs.CreateDirectory(context.Background(), "src/utils")
	if err != nil {
		log.Printf("Error creating directory: %v\n", err)
	}

	// Create a file
	fmt.Println("Creating file...")
	err = fs.CreateFile(context.Background(), "src/utils/helpers.go", `package utils

import "strings"

// StringToUppercase converts a string to uppercase
func StringToUppercase(s string) string {
	return strings.ToUpper(s)
}

// CountWords counts the number of words in a string
func CountWords(s string) int {
	return len(strings.Fields(s))
}
`)
	if err != nil {
		log.Printf("Error creating file: %v\n", err)
	}

	// List directory
	fmt.Println("Listing directory...")
	files, err := fs.ListDir(context.Background(), "src")
	if err != nil {
		log.Printf("Error listing directory: %v\n", err)
	} else {
		for _, f := range files {
			fmt.Printf("  - %s (%d bytes)\n", f.Name, f.Size)
		}
	}

	// Read file
	fmt.Println("Reading file...")
	content, err := fs.ReadFile(context.Background(), "src/utils/helpers.go")
	if err != nil {
		log.Printf("Error reading file: %v\n", err)
	} else {
		fmt.Printf("File content length: %d bytes\n", len(content.Content))
	}

	// Update file
	fmt.Println("Updating file...")
	updatedContent := content.Content + "\n\n// More utility functions..."
	err = fs.WriteFile(context.Background(), "src/utils/helpers.go", updatedContent)
	if err != nil {
		log.Printf("Error updating file: %v\n", err)
	}

	// Rename file
	fmt.Println("Renaming file...")
	err = fs.RenameFile(context.Background(), "src/utils/helpers.go", "src/utils/string_utils.go")
	if err != nil {
		log.Printf("Error renaming file: %v\n", err)
	}

	fmt.Println("Code editor example completed!\n")
}

// ExampleValidation demonstrates the validation functionality
func ExampleValidation() {
	fmt.Println("=== Validation Example ===\n")

	// Create validator
	val := validation.NewValidator()

	// Validate model name
	fmt.Println("Validating model names...")
	val.ValidateModelName("llama2")
	if val.HasErrors() {
		fmt.Printf("Model name validation failed: %v\n", val.FirstError())
	} else {
		fmt.Println("✓ llama2 is valid")
	}

	// Validate file path
	fmt.Println("Validating file paths...")
	val = validation.NewValidator()
	val.ValidateFilePath("src/main.go", ".")
	if val.HasErrors() {
		fmt.Printf("File path validation failed: %v\n", val.FirstError())
	} else {
		fmt.Println("✓ src/main.go is valid")
	}

	// Validate with path traversal attempt
	fmt.Println("Testing path traversal prevention...")
	val = validation.NewValidator()
	val.ValidateFilePath("../../../etc/passwd", ".")
	if val.HasErrors() {
		fmt.Printf("✓ Path traversal blocked: %v\n", val.FirstError())
	}

	// Validate file size
	fmt.Println("Validating file size...")
	val = validation.NewValidator()
	val.ValidateFileSize(1024*1024, 10*1024*1024) // 1MB < 10MB limit
	if val.HasErrors() {
		fmt.Printf("File size validation failed: %v\n", val.FirstError())
	} else {
		fmt.Println("✓ 1MB is within 10MB limit")
	}

	// Validate temperature (for LLM)
	fmt.Println("Validating LLM parameters...")
	val = validation.NewValidator()
	val.ValidateTemperature(0.7)
	if val.HasErrors() {
		fmt.Printf("Temperature validation failed: %v\n", val.FirstError())
	} else {
		fmt.Println("✓ Temperature 0.7 is valid")
	}

	fmt.Println("Validation example completed!\n")
}

// ExampleLocalAgent demonstrates the local agent functionality
func ExampleLocalAgent() {
	fmt.Println("=== Local Agent Example ===\n")

	// Create Ollama client
	client := &api.Client{} // Would be properly initialized with Ollama endpoint

	// Create local agent provider
	agentProvider := agent.NewLocalAgent(client, 5) // Max 5 concurrent tasks

	// Create an agent
	fmt.Println("Creating agent...")
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

	created, err := agentProvider.CreateAgent(context.Background(), codeAgent)
	if err != nil {
		log.Printf("Error creating agent: %v\n", err)
	} else {
		fmt.Printf("✓ Agent created: %s (ID: %s)\n", created.Name, created.ID)
	}

	// List agents
	fmt.Println("Listing agents...")
	agents, err := agentProvider.ListAgents(context.Background())
	if err != nil {
		log.Printf("Error listing agents: %v\n", err)
	} else {
		for _, a := range agents {
			fmt.Printf("  - %s (%s)\n", a.Name, a.ID)
		}
	}

	// Execute a code analysis task
	fmt.Println("Executing code analysis task...")
	if created != nil {
		taskReq := &agent.TaskRequest{
			AgentID: created.ID,
			Type:    "code_analysis",
			Context: "Go REST API handler",
			Prompt:  "Analyze this code for potential issues",
			Files: []agent.TaskFile{
				{
					Path:    "handler.go",
					Content: `package main\n\nfunc HandleRequest(w http.ResponseWriter, r *http.Request) {\n  w.WriteHeader(http.StatusOK)\n}`,
					Type:    "source",
				},
			},
			Timeout: 5 * time.Minute,
		}

		taskResp, err := agentProvider.ExecuteTask(context.Background(), taskReq)
		if err != nil {
			log.Printf("Error executing task: %v\n", err)
		} else {
			fmt.Printf("✓ Task created: %s (Status: %s)\n", taskResp.ID, taskResp.Status)

			// Wait a bit and check status
			time.Sleep(1 * time.Second)
			status, err := agentProvider.GetTaskStatus(context.Background(), taskResp.ID)
			if err != nil {
				log.Printf("Error getting task status: %v\n", err)
			} else {
				fmt.Printf("  Status: %s\n", status.Status)
				if status.Result != "" {
					fmt.Printf("  Result preview: %s...\n", status.Result[:50])
				}
			}
		}
	}

	fmt.Println("Local agent example completed!\n")
}

// ExampleAgentCapabilities demonstrates agent capabilities
func ExampleAgentCapabilities() {
	fmt.Println("=== Agent Capabilities Example ===\n")

	// Create an agent with specific capabilities
	fmt.Println("Creating specialized agents...")

	capabilities := map[string][]agent.AgentCapability{
		"SecurityAuditor": {
			agent.CodeAnalysis,
			agent.SecurityAudit,
			agent.ProjectAnalysis,
		},
		"CodeGenerator": {
			agent.CodeGeneration,
			agent.Testing,
			agent.Documentation,
		},
		"Refactorer": {
			agent.CodeRefactoring,
			agent.PerformanceOptimization,
			agent.CodeAnalysis,
		},
	}

	for name, caps := range capabilities {
		fmt.Printf("%s capabilities:\n", name)
		for _, cap := range caps {
			fmt.Printf("  - %s\n", cap)
		}
		fmt.Println()
	}

	fmt.Println("Agent capabilities example completed!\n")
}

// ExampleConcurrentTasks demonstrates concurrent task execution
func ExampleConcurrentTasks() {
	fmt.Println("=== Concurrent Tasks Example ===\n")

	client := &api.Client{}
	agentProvider := agent.NewLocalAgent(client, 3) // Max 3 concurrent tasks

	// Create an agent
	codeAgent := &agent.Agent{
		Name:  "TaskWorker",
		Type:  agent.LocalAgent,
		Model: "llama2",
		Capabilities: []agent.AgentCapability{
			agent.CodeAnalysis,
			agent.CodeGeneration,
		},
		Config: agent.DefaultAgentConfig(),
	}

	created, _ := agentProvider.CreateAgent(context.Background(), codeAgent)

	// Execute multiple concurrent tasks
	fmt.Println("Executing 5 concurrent tasks with max 3 workers...")
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		taskReq := &agent.TaskRequest{
			AgentID: created.ID,
			Type:    "code_analysis",
			Context: fmt.Sprintf("Task %d context", i),
			Prompt:  fmt.Sprintf("Analyze task %d", i),
			Timeout: 5 * time.Minute,
		}

		resp, _ := agentProvider.ExecuteTask(ctx, taskReq)
		fmt.Printf("  Task %d submitted: %s\n", i, resp.ID)
	}

	// Wait for tasks to complete
	time.Sleep(2 * time.Second)
	fmt.Println("All tasks submitted!")
	fmt.Println("Concurrent tasks example completed!\n")
}

// ExampleErrorHandling demonstrates error handling
func ExampleErrorHandling() {
	fmt.Println("=== Error Handling Example ===\n")

	fs := editor.NewLocalFileSystemManager("./projects")

	// Try to access invalid file path
	fmt.Println("Attempting to read non-existent file...")
	_, err := fs.ReadFile(context.Background(), "non_existent.go")
	if err != nil {
		fmt.Printf("✓ Error caught: %v\n", err)
	}

	// Try to validate invalid path
	fmt.Println("Attempting path traversal...")
	val := validation.NewValidator()
	val.ValidateFilePath("../../sensitive_file", ".")
	if val.HasErrors() {
		fmt.Printf("✓ Validation error caught: %v\n", val.FirstError())
	}

	fmt.Println("Error handling example completed!\n")
}

func main() {
	fmt.Println("\n" + "="*60)
	fmt.Println("OLLAMA CODE EDITOR & AI AGENTS - EXAMPLES")
	fmt.Println("="*60 + "\n")

	// Run all examples
	ExampleValidation()
	ExampleCodeEditor()
	ExampleAgentCapabilities()
	ExampleLocalAgent()
	ExampleConcurrentTasks()
	ExampleErrorHandling()

	fmt.Println("="*60)
	fmt.Println("All examples completed!")
	fmt.Println("="*60 + "\n")
}
