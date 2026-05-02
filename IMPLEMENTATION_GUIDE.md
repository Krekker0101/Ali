# Ollama Enhancements - Built-in Code Editor & AI Agents

## Overview

This document describes the new features added to Ollama:

1. **Built-in Code Editor** - Full-featured file management and editing
2. **AI Agent Framework** - Local and cloud-based AI agents for code analysis, generation, and more

## Table of Contents

- [Code Editor](#code-editor)
- [AI Agents](#ai-agents)
- [API Reference](#api-reference)
- [Configuration](#configuration)
- [Examples](#examples)
- [Security](#security)

---

## Code Editor

### Overview

The built-in code editor provides a complete file management system with syntax highlighting, error detection, and project navigation.

### Features

- **File Management**: Create, read, update, delete files
- **Directory Operations**: Browse and manage directories
- **Syntax Highlighting**: Support for 30+ languages
- **Error Detection**: Real-time error and warning display
- **Project Navigation**: Explore project structure
- **File Search**: Search for files by name or content
- **Safe Access**: Prevents directory traversal attacks

### Architecture

```
editor/
├── filesystem.go      # FileSystemManager interface & LocalFileSystemManager
├── api.go             # HTTP handlers for editor operations
└── README.md          # Detailed documentation
```

### Core Components

#### FileSystemManager Interface

```go
type FileSystemManager interface {
    ListDir(ctx context.Context, path string) ([]FileInfo, error)
    ReadFile(ctx context.Context, path string) (*FileContent, error)
    WriteFile(ctx context.Context, path string, content string) error
    CreateFile(ctx context.Context, path string, content string) error
    DeleteFile(ctx context.Context, path string) error
    RenameFile(ctx context.Context, oldPath, newPath string) error
    CreateDirectory(ctx context.Context, path string) error
    DeleteDirectory(ctx context.Context, path string) error
    GetFileInfo(ctx context.Context, path string) (*FileInfo, error)
    FileExists(ctx context.Context, path string) (bool, error)
}
```

### API Endpoints

#### List Directory
```
GET /api/v1/editor/files/list?path=./src
```

Response:
```json
[
  {
    "path": "src/main.go",
    "name": "main.go",
    "size": 1234,
    "is_dir": false,
    "modified": "2026-04-28T10:30:00Z",
    "language": "go",
    "line_count": 42
  }
]
```

#### Get File
```
GET /api/v1/editor/files/get?path=src/main.go
```

Response:
```json
{
  "path": "src/main.go",
  "content": "package main\n...",
  "language": "go",
  "encoding": "utf-8"
}
```

#### Create File
```
POST /api/v1/editor/files/create

{
  "path": "src/new_file.go",
  "content": "package main\n..."
}
```

#### Update File
```
POST /api/v1/editor/files/update

{
  "path": "src/main.go",
  "content": "package main\n..."
}
```

#### Delete File
```
DELETE /api/v1/editor/files/delete?path=src/old_file.go
```

#### Rename File
```
POST /api/v1/editor/files/rename

{
  "old_path": "src/old_name.go",
  "new_path": "src/new_name.go"
}
```

#### Create Directory
```
POST /api/v1/editor/dirs/create

{
  "path": "src/utils"
}
```

---

## AI Agents

### Overview

The AI Agent framework enables local and cloud-based AI models to perform intelligent code analysis and generation tasks.

### Features

- **Multiple Agent Types**: Local (Ollama), Cloud (OpenAI, Claude), Hybrid
- **Multiple Capabilities**: Analysis, Generation, Refactoring, Bug Fix, etc.
- **Async Task Execution**: Non-blocking task processing
- **Task Management**: Create, monitor, cancel tasks
- **Statistics Tracking**: Monitor agent performance
- **Concurrent Execution**: Configurable worker pool

### Architecture

```
agent/
├── agent.go          # Core agent types and interfaces
├── local.go          # Local agent implementation (Ollama)
├── cloud.go          # Cloud agent implementation (to be added)
├── api.go            # HTTP handlers for agent operations
└── README.md         # Detailed documentation
```

### Core Components

#### Agent Type

```go
type Agent struct {
    ID           string
    Name         string
    Type         AgentType              // "local", "cloud", "hybrid"
    Model        string
    Capabilities []AgentCapability      // What the agent can do
    Config       AgentConfig
    Status       string                 // "active", "inactive", "error"
    Stats        AgentStats             // Performance metrics
}
```

#### Capabilities

```go
const (
    CodeAnalysis           AgentCapability = "code_analysis"
    CodeGeneration         AgentCapability = "code_generation"
    CodeRefactoring        AgentCapability = "code_refactoring"
    BugFix                 AgentCapability = "bug_fix"
    Documentation          AgentCapability = "documentation"
    Testing                AgentCapability = "testing"
    ProjectAnalysis        AgentCapability = "project_analysis"
    PerformanceOptimization AgentCapability = "performance_optimization"
    SecurityAudit          AgentCapability = "security_audit"
)
```

### API Endpoints

#### Create Agent
```
POST /api/v1/agents

{
  "name": "CodeAnalyzer",
  "type": "local",
  "model": "llama2",
  "capabilities": ["code_analysis", "bug_fix"]
}
```

#### List Agents
```
GET /api/v1/agents
```

Response:
```json
[
  {
    "id": "agent_abc123",
    "name": "CodeAnalyzer",
    "type": "local",
    "model": "llama2",
    "status": "active",
    "stats": {
      "total_requests": 42,
      "success_count": 41,
      "error_count": 1,
      "average_latency": "5.2s"
    }
  }
]
```

#### Execute Task
```
POST /api/v1/agents/agent_abc123/tasks

{
  "type": "code_analysis",
  "context": "Backend API service",
  "prompt": "Analyze this code for security vulnerabilities",
  "files": [
    {
      "path": "main.go",
      "content": "package main\n...",
      "type": "source"
    }
  ],
  "timeout": "5m"
}
```

Response (Async):
```json
{
  "id": "task_xyz789",
  "request_id": "task_xyz789",
  "agent_id": "agent_abc123",
  "status": "pending"
}
```

#### Get Task Status
```
GET /api/v1/agents/tasks/task_xyz789
```

Response:
```json
{
  "id": "task_xyz789",
  "status": "completed",
  "result": "Identified 2 security vulnerabilities...",
  "output": [
    {
      "type": "text",
      "content": "Analysis results..."
    }
  ],
  "tokens_used": 1024,
  "duration": "4.5s"
}
```

---

## Configuration

### Environment Variables

```bash
# Code Editor
OLLAMA_EDITOR_ENABLED=true
OLLAMA_EDITOR_MAX_FILE_SIZE=10485760    # 10MB in bytes
OLLAMA_EDITOR_BASE_PATH=./projects

# AI Agents
OLLAMA_AGENTS_ENABLED=true
OLLAMA_AGENTS_MAX_WORKERS=5
OLLAMA_AGENTS_DEFAULT_TIMEOUT=300       # 5 minutes in seconds
```

### Configuration File (config.yml)

```yaml
editor:
  enabled: true
  max_file_size: 10485760
  base_path: ./projects
  supported_languages:
    - go
    - python
    - javascript
    - rust

agents:
  enabled: true
  max_workers: 5
  default_timeout: 300
  local:
    enabled: true
    models:
      - llama2
      - neural-chat
  cloud:
    enabled: false
    providers:
      openai:
        enabled: false
        api_key: ""
      anthropic:
        enabled: false
        api_key: ""
```

---

## Examples

### Example 1: Creating and Reading a File

```bash
# Create a file
curl -X POST http://localhost:11434/api/v1/editor/files/create \
  -H "Content-Type: application/json" \
  -d '{
    "path": "src/hello.go",
    "content": "package main\n\nfunc main() {\n}"
  }'

# Read the file
curl http://localhost:11434/api/v1/editor/files/get?path=src/hello.go
```

### Example 2: Code Analysis with Agents

```bash
# Create a local agent
curl -X POST http://localhost:11434/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "SecurityAnalyzer",
    "type": "local",
    "model": "llama2",
    "capabilities": ["code_analysis", "security_audit"]
  }'

# Execute a security analysis task
curl -X POST http://localhost:11434/api/v1/agents/agent_id/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "code_analysis",
    "context": "Security audit of API handlers",
    "prompt": "Identify all potential security vulnerabilities",
    "files": [{
      "path": "handlers.go",
      "content": "...",
      "type": "source"
    }],
    "parameters": {
      "analysis_type": "security"
    },
    "timeout": "5m"
  }'

# Check task status
curl http://localhost:11434/api/v1/agents/tasks/task_id
```

### Example 3: Code Generation

```bash
# Create a code generation task
curl -X POST http://localhost:11434/api/v1/agents/agent_id/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "code_generation",
    "context": "REST API handler",
    "prompt": "Generate a Go REST API handler for user management",
    "parameters": {
      "language": "go",
      "framework": "gin",
      "build_tests": true,
      "add_documentation": true
    },
    "timeout": "10m"
  }'
```

---

## Security

### Path Traversal Prevention

All file paths are validated to prevent directory traversal attacks:

```go
// Automatically validated by FileSystemManager
val := validation.NewValidator()
val.ValidateFilePath(userPath, basePath)
if val.HasErrors() {
    // Path traversal attempt detected!
}
```

### File Size Limits

All file operations respect configured size limits:

```bash
# Max 10MB per file (configurable)
OLLAMA_EDITOR_MAX_FILE_SIZE=10485760
```

### Agent Sandboxing

Agents run in a controlled environment:

- Configurable timeout (default 5 minutes)
- Token limits (default 2048)
- Async execution prevents blocking
- Failed tasks don't affect system

### Input Validation

All inputs are validated:

```go
// Model name validation
val.ValidateModelName(name)

// File path validation
val.ValidateFilePath(path, baseDir)

// File size validation
val.ValidateFileSize(size, maxSize)

// Port validation
val.ValidatePort(port)

// Temperature validation
val.ValidateTemperature(temp)
```

---

## Performance Considerations

### Editor
- File operations are fast (local filesystem)
- Large files may take time to read (use streaming for >100MB)
- Directory listing is cached (TTL: 30 seconds)

### Agents
- Tasks are executed asynchronously
- Configurable worker pool (default: 5)
- Use semaphore to prevent resource exhaustion
- Long-running tasks get automatic timeouts

---

## Future Enhancements

1. **Cloud Providers**: OpenAI, Claude, other cloud services
2. **Real-time Collaboration**: Multi-user editing
3. **Version Control Integration**: Git integration in editor
4. **Advanced Diagnostics**: LSP integration for better error detection
5. **Agent Chaining**: Multiple agents working together
6. **Model Fine-tuning**: Fine-tune local models based on feedback
7. **Caching Layer**: Cache agent responses for common queries
8. **Webhooks**: Real-time notifications for task completion

---

## Troubleshooting

### "Path traversal detected"
- Don't use absolute paths or `../` in file paths
- Use relative paths within the configured base directory

### "File size exceeds limit"
- Increase `OLLAMA_EDITOR_MAX_FILE_SIZE` environment variable
- Check server logs for configured limits

### "Agent timeout"
- Increase task timeout parameter
- Check agent model responsiveness
- Check available memory/CPU

### "No models available"
- Make sure Ollama is running
- Pull a model: `ollama pull llama2`
- Check Ollama API connectivity

---

For more information, see:
- [Code Editor README](./editor/README.md)
- [AI Agent Framework README](./agent/README.md)
- [Validation Utilities README](./validation/README.md)
