# Validation Package

## Overview

The validation package provides comprehensive input validation for all user inputs, preventing security vulnerabilities and malformed data.

## Features

- **Path Security**: Prevents directory traversal attacks
- **File Validation**: File size, name, and extension checks
- **LLM Parameters**: Temperature, TopP, TopK validation
- **Network**: Port validation
- **Model Names**: Model name format validation
- **Fluent API**: Method chaining for easy validation

## Usage

### Basic Validation

```go
import "github.com/ollama/ollama/validation"

// Create validator
val := validation.NewValidator()

// Validate model name
val.ValidateModelName("llama2")

// Check errors
if val.HasErrors() {
    fmt.Println(val.FirstError())
}
```

### Path Validation with Security

```go
// Validates path is within basePath (prevents traversal)
val := validation.NewValidator()
val.ValidateFilePath("src/main.go", ".")

// These will be rejected:
// - "../../../etc/passwd" (traversal)
// - "/etc/passwd" (absolute path)
```

### Fluent API

```go
val := validation.NewValidator()
err := val.
    ValidateModelName("llama2").
    ValidatePort(8080).
    ValidateTemperature(0.7).
    ValidateFileSize(1024*1024, 10*1024*1024).
    FirstError()

if err != nil {
    fmt.Println(err)
}
```

## Validation Methods

### ValidateModelName(name string)
- Validates LLM model names
- Checks format: alphanumeric, hyphens, underscores
- Max length: 100 characters

### ValidateFilePath(path, basePath string)
- Prevents directory traversal
- Ensures path is within basePath
- Normalizes path using filepath.Clean

### ValidateFileSize(size, maxSize int64)
- Checks file doesn't exceed max size
- Validates size is positive

### ValidateFileName(name string)
- Validates individual file names
- Prevents problematic characters
- Max length: 255 characters

### ValidateFileExtension(filename, allowedExt string)
- Validates file extension
- Case-insensitive comparison
- Multiple extensions can be comma-separated

### ValidatePort(port int)
- Validates network port
- Range: 1-65535

### ValidateTimeout(seconds int)
- Validates timeout duration
- Range: 1-3600 seconds (1 hour max)

### ValidateNumCtx(numCtx int)
- Validates context tokens
- Range: 128-131072

### ValidateTemperature(temp float64)
- Validates LLM temperature parameter
- Range: 0.0-2.0

### ValidateTopP(topP float64)
- Validates top-p sampling parameter
- Range: 0.0-1.0

### ValidateTopK(topK int)
- Validates top-k sampling parameter
- Range: 0-200

## Error Handling

```go
val := validation.NewValidator()
val.ValidatePort(99999).ValidatePort(-1)

// Get first error
err := val.FirstError()

// Get all errors
errors := val.Errors()

// Check if has errors
if val.HasErrors() {
    // Handle errors
}

// Get formatted error string
fmt.Println(val.String())
```

## Security Features

### Path Traversal Prevention
- Uses `filepath.Clean` to normalize paths
- Detects `..` patterns
- Prevents absolute paths
- Validates path is within base directory

### Input Validation
- Whitelist-based character validation
- Length limits for all strings
- Range validation for numeric parameters
- Format validation using regex

## Performance

- O(1) or O(n) where n is input length
- No external dependencies (stdlib only)
- Lazy validation (only validates called methods)
- Early error returns

## Error Messages

All error messages are clear and actionable:

```
"invalid model name: expected alphanumeric, got 'my-model-!''"
"path traversal detected: ../../../etc/passwd"
"file size exceeds limit: 12582912 > 10485760"
"port out of range: 99999 (valid: 1-65535)"
"temperature out of range: 2.5 (valid: 0.0-2.0)"
```

## Thread Safety

The validator is not thread-safe. Create separate validator instances for concurrent validation:

```go
// DO NOT do this in concurrent code:
sharedValidator := validation.NewValidator()

// Instead, create per-goroutine validators:
go func() {
    val := validation.NewValidator()
    val.ValidatePort(8080)
}()
```

## Examples

See [examples_new_features.go](../examples_new_features.go) for complete examples.

## Related

- [Code Editor](../editor/README.md) - Uses validation for file operations
- [AI Agents](../agent/README.md) - Uses validation for agent configuration
