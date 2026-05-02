package editor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo contains metadata about a file
type FileInfo struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	IsDir     bool      `json:"is_dir"`
	Modified  time.Time `json:"modified"`
	Language  string    `json:"language"`
	LineCount int       `json:"line_count"`
}

// FileContent represents file content with metadata
type FileContent struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
	Encoding string `json:"encoding"`
}

// EditOperation represents a code edit operation
type EditOperation struct {
	Path   string `json:"path"`
	Action string `json:"action"` // "create", "update", "delete", "rename"
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	NewPath string `json:"new_path,omitempty"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

// FileSystemManager manages file operations
type FileSystemManager interface {
	// ListDir lists files in a directory
	ListDir(ctx context.Context, path string) ([]FileInfo, error)
	
	// ReadFile reads file content
	ReadFile(ctx context.Context, path string) (*FileContent, error)
	
	// WriteFile writes file content
	WriteFile(ctx context.Context, path string, content string) error
	
	// CreateFile creates a new file
	CreateFile(ctx context.Context, path string, content string) error
	
	// DeleteFile deletes a file
	DeleteFile(ctx context.Context, path string) error
	
	// RenameFile renames a file
	RenameFile(ctx context.Context, oldPath, newPath string) error
	
	// CreateDirectory creates a directory
	CreateDirectory(ctx context.Context, path string) error
	
	// DeleteDirectory deletes a directory
	DeleteDirectory(ctx context.Context, path string) error
	
	// GetFileInfo gets file information
	GetFileInfo(ctx context.Context, path string) (*FileInfo, error)
	
	// FileExists checks if a file exists
	FileExists(ctx context.Context, path string) (bool, error)
}

// SyntaxHighlighter provides syntax highlighting
type SyntaxHighlighter interface {
	// Highlight returns highlighted code
	Highlight(code string, language string) (string, error)
	
	// GetLanguage determines language from file extension
	GetLanguage(filename string) string
	
	// GetSupportedLanguages returns list of supported languages
	GetSupportedLanguages() []string
}

// DiagnosticsCollector collects diagnostics (errors, warnings)
type DiagnosticsCollector interface {
	// CollectDiagnostics collects diagnostics for files
	CollectDiagnostics(ctx context.Context, paths []string) (map[string][]Diagnostic, error)
}

// Diagnostic represents an error or warning in code
type Diagnostic struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Severity  string `json:"severity"` // "error", "warning", "info"
	Message   string `json:"message"`
	Code      string `json:"code,omitempty"`
	Source    string `json:"source,omitempty"` // "linter", "compiler", etc.
}

// EditorState represents the state of the editor
type EditorState struct {
	OpenFiles   map[string]*FileContent `json:"open_files"`
	CurrentFile string                  `json:"current_file"`
	Diagnostics map[string][]Diagnostic `json:"diagnostics"`
	LastSync    time.Time               `json:"last_sync"`
}

// EditorService provides editor operations
type EditorService interface {
	// GetFileContent gets file content with syntax info
	GetFileContent(ctx context.Context, path string) (*FileContent, error)
	
	// UpdateFile updates a file and returns diagnostics
	UpdateFile(ctx context.Context, path string, content string) ([]Diagnostic, error)
	
	// ApplyEdit applies an edit operation
	ApplyEdit(ctx context.Context, op EditOperation) (*EditOperation, error)
	
	// GetDiagnostics gets diagnostics for a file
	GetDiagnostics(ctx context.Context, path string) ([]Diagnostic, error)
	
	// GetDiagnosticsForPath gets diagnostics for all files in path
	GetDiagnosticsForDir(ctx context.Context, path string) (map[string][]Diagnostic, error)
	
	// Format formats code
	Format(ctx context.Context, code string, language string) (string, error)
	
	// GetCompletions gets code completions
	GetCompletions(ctx context.Context, path string, line int, column int) ([]string, error)
	
	// GetDefinition gets definition location
	GetDefinition(ctx context.Context, path string, line int, column int) (*FileInfo, error)
	
	// FindReferences finds all references to symbol
	FindReferences(ctx context.Context, path string, line int, column int) ([]*FileInfo, error)
	
	// Search searches for text in files
	Search(ctx context.Context, query string, path string) (map[string][]SearchResult, error)
	
	// GetProjectStructure gets project structure
	GetProjectStructure(ctx context.Context, rootPath string) (*ProjectStructure, error)
}

// SearchResult represents a search result
type SearchResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Context string `json:"context"`
}

// ProjectStructure represents the project structure
type ProjectStructure struct {
	Root  string        `json:"root"`
	Files []FileInfo    `json:"files"`
	Dirs  []string      `json:"dirs"`
	Total int           `json:"total"`
	Stats ProjectStats  `json:"stats"`
}

// ProjectStats contains project statistics
type ProjectStats struct {
	TotalFiles   int            `json:"total_files"`
	TotalDirs    int            `json:"total_dirs"`
	TotalSize    int64          `json:"total_size"`
	FilesByType  map[string]int `json:"files_by_type"`
	LargestFiles []string       `json:"largest_files"`
}

// LocalFileSystemManager implements FileSystemManager using local filesystem
type LocalFileSystemManager struct {
	basePath string
	maxSize  int64
}

// NewLocalFileSystemManager creates a new local filesystem manager
func NewLocalFileSystemManager(basePath string, maxSizeBytes int64) *LocalFileSystemManager {
	return &LocalFileSystemManager{
		basePath: basePath,
		maxSize:  maxSizeBytes,
	}
}

// ListDir lists files in a directory
func (m *LocalFileSystemManager) ListDir(ctx context.Context, path string) ([]FileInfo, error) {
	fullPath := filepath.Join(m.basePath, path)
	
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}
	
	var files []FileInfo
	for _, entry := range entries {
		info, _ := entry.Info()
		if info == nil {
			continue
		}
		
		files = append(files, FileInfo{
			Path:     filepath.Join(path, entry.Name()),
			Name:     entry.Name(),
			Size:     info.Size(),
			IsDir:    entry.IsDir(),
			Modified: info.ModTime(),
			Language: getLanguageFromName(entry.Name()),
		})
	}
	
	return files, nil
}

// ReadFile reads file content
func (m *LocalFileSystemManager) ReadFile(ctx context.Context, path string) (*FileContent, error) {
	fullPath := filepath.Join(m.basePath, path)
	
	// Verify path is within basePath (security check)
	cleanBase := filepath.Clean(m.basePath)
	cleanFull := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanFull, cleanBase) {
		return nil, os.ErrPermission
	}
	
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	
	return &FileContent{
		Path:     path,
		Content:  string(content),
		Language: getLanguageFromName(filepath.Base(path)),
		Encoding: "utf-8",
	}, nil
}

// WriteFile writes file content
func (m *LocalFileSystemManager) WriteFile(ctx context.Context, path string, content string) error {
	fullPath := filepath.Join(m.basePath, path)
	
	// Verify path is within basePath (security check)
	cleanBase := filepath.Clean(m.basePath)
	cleanFull := filepath.Clean(fullPath)
	if !filepath.HasPrefix(cleanFull, cleanBase) {
		return os.ErrPermission
	}
	
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// CreateFile creates a new file
func (m *LocalFileSystemManager) CreateFile(ctx context.Context, path string, content string) error {
	fullPath := filepath.Join(m.basePath, path)
	
	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// DeleteFile deletes a file
func (m *LocalFileSystemManager) DeleteFile(ctx context.Context, path string) error {
	fullPath := filepath.Join(m.basePath, path)
	
	// Verify path is within basePath (security check)
	cleanBase := filepath.Clean(m.basePath)
	cleanFull := filepath.Clean(fullPath)
	if !filepath.HasPrefix(cleanFull, cleanBase) {
		return os.ErrPermission
	}
	
	return os.Remove(fullPath)
}

// RenameFile renames a file
func (m *LocalFileSystemManager) RenameFile(ctx context.Context, oldPath, newPath string) error {
	fullOld := filepath.Join(m.basePath, oldPath)
	fullNew := filepath.Join(m.basePath, newPath)
	
	// Verify paths are within basePath
	cleanBase := filepath.Clean(m.basePath)
	if !strings.HasPrefix(filepath.Clean(fullOld), cleanBase) || !strings.HasPrefix(filepath.Clean(fullNew), cleanBase) {
		return os.ErrPermission
	}
	
	return os.Rename(fullOld, fullNew)
}

// CreateDirectory creates a directory
func (m *LocalFileSystemManager) CreateDirectory(ctx context.Context, path string) error {
	fullPath := filepath.Join(m.basePath, path)
	return os.MkdirAll(fullPath, 0755)
}

// DeleteDirectory deletes a directory
func (m *LocalFileSystemManager) DeleteDirectory(ctx context.Context, path string) error {
	fullPath := filepath.Join(m.basePath, path)
	
	// Verify path is within basePath
	cleanBase := filepath.Clean(m.basePath)
	cleanFull := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanFull, cleanBase) {
		return os.ErrPermission
	}
	
	return os.RemoveAll(fullPath)
}

// GetFileInfo gets file information
func (m *LocalFileSystemManager) GetFileInfo(ctx context.Context, path string) (*FileInfo, error) {
	fullPath := filepath.Join(m.basePath, path)
	
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	
	return &FileInfo{
		Path:     path,
		Name:     info.Name(),
		Size:     info.Size(),
		IsDir:    info.IsDir(),
		Modified: info.ModTime(),
		Language: getLanguageFromName(filepath.Base(path)),
	}, nil
}

// FileExists checks if a file exists
func (m *LocalFileSystemManager) FileExists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(m.basePath, path)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// getLanguageFromName determines language from file name
func getLanguageFromName(filename string) string {
	ext := filepath.Ext(filename)
	
	languageMap := map[string]string{
		".go":       "go",
		".py":       "python",
		".js":       "javascript",
		".ts":       "typescript",
		".jsx":      "jsx",
		".tsx":      "tsx",
		".java":     "java",
		".cpp":      "cpp",
		".c":        "c",
		".h":        "c",
		".hpp":      "cpp",
		".rs":       "rust",
		".rb":       "ruby",
		".php":      "php",
		".swift":    "swift",
		".kt":       "kotlin",
		".scala":    "scala",
		".sh":       "bash",
		".bash":     "bash",
		".zsh":      "zsh",
		".fish":     "fish",
		".ps1":      "powershell",
		".json":     "json",
		".yaml":     "yaml",
		".yml":      "yaml",
		".toml":     "toml",
		".ini":      "ini",
		".xml":      "xml",
		".html":     "html",
		".htm":      "html",
		".css":      "css",
		".scss":     "scss",
		".less":     "less",
		".md":       "markdown",
		".txt":      "text",
		".sql":      "sql",
		".lua":      "lua",
		".r":        "r",
		".R":        "r",
		".m":        "objective-c",
		".mm":       "objective-cpp",
		".groovy":   "groovy",
		".gradle":   "gradle",
		".Dockerfile": "dockerfile",
		".dockerfile": "dockerfile",
	}
	
	if lang, ok := languageMap[ext]; ok {
		return lang
	}
	
	return "text"
}
