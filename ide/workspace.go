package ide

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxFileSize = 2 * 1024 * 1024
	maxTreeEntries     = 2500
	maxSearchResults   = 300
)

var skippedTreeDirs = map[string]bool{
	".git":        true,
	".idea":       true,
	".vscode":     false,
	"node_modules": true,
	"dist":        true,
	"build":       true,
	"tmp":         true,
	"vendor":      true,
}

type Workspace struct {
	mu          sync.RWMutex
	root        string
	openedAt    time.Time
	maxFileSize int64
}

func NewWorkspace(root string, maxFileSize int64) (*Workspace, error) {
	if maxFileSize <= 0 {
		maxFileSize = defaultMaxFileSize
	}

	w := &Workspace{maxFileSize: maxFileSize}
	if err := w.Open(root); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Workspace) Open(root string) error {
	if strings.TrimSpace(root) == "" {
		root = "."
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("workspace root is not a directory: %s", root)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.root = filepath.Clean(resolved)
	w.openedAt = time.Now()
	return nil
}

func (w *Workspace) State() WorkspaceState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return WorkspaceState{
		Root:     w.root,
		Name:     filepath.Base(w.root),
		OpenedAt: w.openedAt,
	}
}

func (w *Workspace) ListDir(ctx context.Context, path string) ([]FileNode, error) {
	full, rel, err := w.resolve(path, true)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", path)
	}

	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}

	nodes := make([]FileNode, 0, len(entries))
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		childRel := entry.Name()
		if rel != "." {
			childRel = filepath.Join(rel, entry.Name())
		}

		nodes = append(nodes, fileNodeFromInfo(childRel, info))
	}

	sortNodes(nodes)
	return nodes, nil
}

func (w *Workspace) Tree(ctx context.Context, path string, depth int) ([]FileNode, error) {
	if depth <= 0 {
		depth = 3
	}
	if depth > 8 {
		depth = 8
	}

	full, rel, err := w.resolve(path, true)
	if err != nil {
		return nil, err
	}

	count := 0
	nodes, err := w.treeAt(ctx, full, rel, depth, &count)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func (w *Workspace) ReadFile(ctx context.Context, path string) (*FileContent, error) {
	full, rel, err := w.resolve(path, true)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("cannot read a directory: %s", path)
	}
	if info.Size() > w.maxFileSize {
		return nil, fmt.Errorf("file is larger than the IDE limit (%d bytes)", w.maxFileSize)
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	if looksBinary(data) {
		return nil, fmt.Errorf("binary files are not opened in the IDE")
	}

	return &FileContent{
		Path:     filepath.ToSlash(rel),
		Content:  string(data),
		Language: languageForFile(rel),
		Encoding: "utf-8",
		Size:     info.Size(),
	}, nil
}

func (w *Workspace) WriteFile(ctx context.Context, path string, content string) error {
	full, _, err := w.resolve(path, false)
	if err != nil {
		return err
	}
	if int64(len(content)) > w.maxFileSize {
		return fmt.Errorf("content is larger than the IDE limit (%d bytes)", w.maxFileSize)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0o644)
}

func (w *Workspace) CreateFile(ctx context.Context, path string, content string) error {
	full, _, err := w.resolve(path, false)
	if err != nil {
		return err
	}
	if _, err := os.Stat(full); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return w.WriteFile(ctx, path, content)
}

func (w *Workspace) DeleteFile(ctx context.Context, path string) error {
	full, _, err := w.resolve(path, true)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	info, err := os.Stat(full)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("delete file refuses directories: %s", path)
	}
	return os.Remove(full)
}

func (w *Workspace) Search(ctx context.Context, query string, path string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	root, _, err := w.resolve(path, true)
	if err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)
	results := make([]SearchResult, 0)
	err = filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if len(results) >= maxSearchResults {
			return fs.SkipAll
		}
		if entry.IsDir() {
			if skippedTreeDirs[entry.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		info, err := entry.Info()
		if err != nil || info.Size() > w.maxFileSize {
			return nil
		}

		data, err := os.ReadFile(current)
		if err != nil || looksBinary(data) {
			return nil
		}

		rel, err := w.relative(current)
		if err != nil {
			return nil
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		lineNumber := 0
		for scanner.Scan() {
			lineNumber++
			line := scanner.Text()
			column := strings.Index(strings.ToLower(line), lowerQuery)
			if column >= 0 {
				results = append(results, SearchResult{
					Path:    rel,
					Line:    lineNumber,
					Column:  column + 1,
					Preview: strings.TrimSpace(line),
				})
				if len(results) >= maxSearchResults {
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (w *Workspace) resolve(path string, mustExist bool) (string, string, error) {
	rel, err := cleanRelative(path)
	if err != nil {
		return "", "", err
	}

	w.mu.RLock()
	root := w.root
	w.mu.RUnlock()

	if root == "" {
		return "", "", fmt.Errorf("workspace is not open")
	}

	candidate := filepath.Join(root, rel)
	if resolvedTarget, err := filepath.EvalSymlinks(candidate); err == nil {
		candidate = resolvedTarget
	} else if mustExist {
		return "", "", err
	} else {
		parent := filepath.Dir(candidate)
		if resolvedParent, err := filepath.EvalSymlinks(parent); err == nil {
			candidate = filepath.Join(resolvedParent, filepath.Base(candidate))
		}
	}

	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", "", err
	}
	if !isInside(root, abs) {
		return "", "", os.ErrPermission
	}
	return abs, rel, nil
}

func (w *Workspace) relative(full string) (string, error) {
	w.mu.RLock()
	root := w.root
	w.mu.RUnlock()

	rel, err := filepath.Rel(root, full)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func (w *Workspace) treeAt(ctx context.Context, full string, rel string, depth int, count *int) ([]FileNode, error) {
	if *count >= maxTreeEntries || depth <= 0 {
		return nil, nil
	}

	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}

	nodes := make([]FileNode, 0, len(entries))
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if *count >= maxTreeEntries {
			break
		}
		if entry.IsDir() && skippedTreeDirs[entry.Name()] {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		childRel := entry.Name()
		if rel != "." {
			childRel = filepath.Join(rel, entry.Name())
		}

		node := fileNodeFromInfo(childRel, info)
		*count = *count + 1
		if info.IsDir() {
			children, err := w.treeAt(ctx, filepath.Join(full, entry.Name()), childRel, depth-1, count)
			if err == nil {
				node.Children = children
			}
		}
		nodes = append(nodes, node)
	}
	sortNodes(nodes)
	return nodes, nil
}

func cleanRelative(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		path = "."
	}

	path = filepath.FromSlash(strings.TrimSpace(path))
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute paths are only allowed when opening a workspace")
	}

	clean := filepath.Clean(path)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", os.ErrPermission
	}
	return clean, nil
}

func isInside(root string, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (!filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func fileNodeFromInfo(path string, info os.FileInfo) FileNode {
	nodeType := "file"
	if info.IsDir() {
		nodeType = "directory"
	}
	return FileNode{
		Path:     filepath.ToSlash(path),
		Name:     info.Name(),
		Type:     nodeType,
		Size:     info.Size(),
		Modified: info.ModTime(),
		Language: languageForFile(path),
	}
}

func sortNodes(nodes []FileNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Type != nodes[j].Type {
			return nodes[i].Type == "directory"
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
}

func looksBinary(data []byte) bool {
	sample := data
	if len(sample) > 8000 {
		sample = sample[:8000]
	}
	return bytes.IndexByte(sample, 0) >= 0
}

func languageForFile(path string) string {
	name := strings.ToLower(filepath.Base(path))
	if name == "dockerfile" {
		return "dockerfile"
	}
	if name == "makefile" {
		return "makefile"
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".mod", ".sum":
		return "go"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".tsx":
		return "tsx"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".md", ".mdx":
		return "markdown"
	case ".html", ".htm":
		return "html"
	case ".css", ".scss", ".less":
		return "css"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cc", ".cpp", ".cxx", ".hpp":
		return "cpp"
	case ".sh", ".bash", ".zsh":
		return "shell"
	case ".ps1":
		return "powershell"
	case ".sql":
		return "sql"
	case ".xml":
		return "xml"
	default:
		return "text"
	}
}
