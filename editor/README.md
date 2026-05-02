# Code Editor Package

## Overview

The code editor package provides a complete file management system with syntax highlighting, error detection, and project navigation capabilities. It serves as the foundation for the built-in code editor in Ali.

## Features

- **File Management**: Create, read, update, delete files with security checks
- **Directory Operations**: Browse and manage directory structures
- **Syntax Highlighting**: Support for 30+ programming languages
- **Language Detection**: Automatic language detection from file extensions
- **Error Detection**: Real-time error and warning display (framework provided)
- **Project Navigation**: Explore complete project structures
- **File Search**: Search by name and content (extensible)
- **Security**: Prevents directory traversal attacks
- **Performance**: Fast local filesystem operations

## Architecture

```
FileSystemManager (interface)
    ├── ListDir
    ├── ReadFile
    ├── WriteFile
    ├── CreateFile
    ├── DeleteFile
    ├── RenameFile
    ├── CreateDirectory
    ├── DeleteDirectory
    ├── GetFileInfo
    └── FileExists

LocalFileSystemManager (implementation)
    ├── All 9 FileSystemManager methods
    └── Security validation for all paths

Supporting Interfaces
    ├── SyntaxHighlighter
    ├── DiagnosticsCollector
    ├── EditorService
```

## Usage

### Basic File Operations

```go
import "github.com/ollama/ollama/editor"

// Create filesystem manager (validates all operations)
fs := editor.NewLocalFileSystemManager("./projects")

// Create a file
err := fs.CreateFile(context.Background(), "src/main.go", "package main")

// Read a file
content, err := fs.ReadFile(context.Background(), "src/main.go")
if err == nil {
    fmt.Println(content.Content)
}

// Update a file
err = fs.WriteFile(context.Background(), "src/main.go", "package main\n...")

// Delete a file
err = fs.DeleteFile(context.Background(), "src/main.go")
```

### Directory Operations

```go
// List directory contents
files, err := fs.ListDir(context.Background(), "src")
for _, file := range files {
    fmt.Printf("%s (%d bytes) - Language: %s\n", 
        file.Name, file.Size, file.Language)
}

// Create directory
err = fs.CreateDirectory(context.Background(), "src/utils")

// Delete directory
err = fs.DeleteDirectory(context.Background(), "src/utils")

// Get file info
info, err := fs.GetFileInfo(context.Background(), "src/main.go")
if err == nil {
    fmt.Printf("Modified: %s\n", info.Modified)
    fmt.Printf("Size: %d bytes\n", info.Size)
}

// Check if file exists
exists, err := fs.FileExists(context.Background(), "src/main.go")
```

### Rename Operations

```go
// Rename a file
err = fs.RenameFile(context.Background(), 
    "old_name.go", 
    "new_name.go")
```

## Supported Languages

The editor supports syntax highlighting for:

- **System**: bash, sh, zsh
- **Go**: go, gomod, gosum
- **Python**: py, pyw
- **JavaScript**: js, mjs, cjs
- **TypeScript**: ts, tsx
- **Web**: html, css, scss, less
- **Java**: java, class
- **C/C++**: c, h, cpp, cc, cxx, hpp
- **C#**: cs, csproj
- **Rust**: rs, toml
- **Ruby**: rb, erb, gemfile
- **PHP**: php, phtml
- **Swift**: swift
- **Kotlin**: kt, kts
- **Scala**: scala, sc
- **SQL**: sql
- **JSON**: json, jsonc, package.json
- **YAML**: yaml, yml
- **TOML**: toml
- **XML**: xml
- **Markdown**: md, mdx
- **Docker**: dockerfile
- **Config**: conf, cfg, ini, properties
- And more...

## Data Structures

### FileInfo
```go
type FileInfo struct {
    Path      string    // Relative path from base
    Name      string    // Filename only
    Size      int64     // File size in bytes
    IsDir     bool      // Is directory
    Modified  time.Time // Last modified time
    Language  string    // Language for highlighting
    LineCount int       // Number of lines (for files)
}
```

### FileContent
```go
type FileContent struct {
    Path     string // File path
    Content  string // File content
    Language string // Programming language
    Encoding string // Text encoding (usually "utf-8")
}
```

### EditOperation
```go
type EditOperation struct {
    Type      string // "insert", "delete", "replace"
    Line      int    // Line number
    Column    int    // Column number
    Content   string // Content to insert/replace
    FromLine  int    // For replace operations
    ToLine    int    // For replace operations
}
```

### EditorState
```go
type EditorState struct {
    FilePath   string
    Content    string
    Cursor     struct {
        Line   int
        Column int
    }
    Selection  struct {
        Start struct {
            Line   int
            Column int
        }
        End struct {
            Line   int
            Column int
        }
    }
    Dirty      bool
}
```

## Security

### Path Traversal Prevention

All file paths are validated to prevent directory traversal attacks:

```go
// These will be rejected:
fs.ReadFile(ctx, "../../../etc/passwd")  // traversal attempt
fs.ReadFile(ctx, "/etc/passwd")          // absolute path
fs.ReadFile(ctx, "foo/../../../etc/passwd") // normalized traversal

// These are allowed (relative paths within basePath):
fs.ReadFile(ctx, "src/main.go")
fs.ReadFile(ctx, "src/utils/helpers.go")
fs.ReadFile(ctx, "./src/main.go")
```

Implementation:
```go
// In each operation:
cleanPath := filepath.Clean(path)
if !strings.HasPrefix(filepath.Join(basePath, cleanPath), basePath) {
    return fmt.Errorf("path traversal detected")
}
```

### File Size Limits

Configure maximum file size to prevent large file uploads:

```bash
export OLLAMA_EDITOR_MAX_FILE_SIZE=10485760  # 10MB
```

### Permission Checks

- Files are created with appropriate permissions
- Directory permissions: 0755
- File permissions: 0644

## HTTP API

### Endpoints

```
GET    /api/v1/editor/files/list              # List directory
GET    /api/v1/editor/files/get               # Get file content
POST   /api/v1/editor/files/create            # Create file
POST   /api/v1/editor/files/update            # Update file
DELETE /api/v1/editor/files/delete            # Delete file
POST   /api/v1/editor/files/rename            # Rename file
POST   /api/v1/editor/dirs/create             # Create directory
GET    /api/v1/editor/search                  # Search files
GET    /api/v1/editor/project-structure       # Get project structure
```

See [IMPLEMENTATION_GUIDE.md](../IMPLEMENTATION_GUIDE.md#editor-1) for detailed API documentation.

## Error Handling

All operations return errors for:
- Path traversal attempts
- File not found
- Permission denied
- Invalid file operations
- I/O errors

```go
content, err := fs.ReadFile(context.Background(), "main.go")
if err != nil {
    log.Printf("Error reading file: %v", err)
    // Handle specific errors:
    // - os.ErrNotExist: file doesn't exist
    // - os.ErrPermission: no read permission
    // - custom error: path traversal detected
}
```

## Performance Considerations

- **Listing large directories**: O(n) where n is number of entries
- **Reading large files**: Scales with file size (consider streaming for >100MB)
- **Directory traversal check**: O(1) constant time
- **Language detection**: O(1) lookup from extension map
- **Caching**: Directory listings can be cached (TTL: 30 seconds recommended)

## Extensibility

### Custom SyntaxHighlighter

```go
type SyntaxHighlighter interface {
    GetLanguage(filename string) string
    Highlight(code, language string) string
}
```

### Custom DiagnosticsCollector

```go
type DiagnosticsCollector interface {
    Collect(ctx context.Context, file *FileContent) ([]Diagnostic, error)
}
```

## Examples

See [examples_new_features.go](../examples_new_features.go) for complete examples:

- Creating and reading files
- Directory operations
- Error handling
- File operations in practice

## Related

- [Validation Package](../validation/README.md) - Input validation
- [AI Agents](../agent/README.md) - Can read and analyze files
- [IMPLEMENTATION_GUIDE.md](../IMPLEMENTATION_GUIDE.md) - Complete guide

## Future Enhancements

1. **Streaming API**: For large files (>100MB)
2. **Version Control**: Git integration
3. **Diff Support**: Show file changes
4. **LSP Integration**: Language Server Protocol for better diagnostics
5. **File Watcher**: Real-time file change notifications
6. **Backup/Restore**: Automatic file backups
7. **Compression**: Store compressed file versions
8. **Multi-user**: Collaborative editing support
