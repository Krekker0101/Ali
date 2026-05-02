package ide

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const maxAgentContextChars = 60000

type Service struct {
	workspace *Workspace
	providers *ProviderRegistry
	mu        sync.RWMutex
	settings  Settings
}

func NewService(root string) (*Service, error) {
	workspace, err := NewWorkspace(root, defaultMaxFileSize)
	if err != nil {
		return nil, err
	}

	return &Service{
		workspace: workspace,
		providers: NewProviderRegistry(),
		settings:  loadPersistedSettings(DefaultSettings()),
	}, nil
}

func DefaultSettings() Settings {
	return Settings{
		Theme: ThemeSettings{
			Mode: "dark",
			Colors: map[string]string{
				"background": "#0f1115",
				"panel":      "rgba(24, 26, 32, 0.68)",
				"panel2":     "rgba(36, 39, 48, 0.62)",
				"text":       "#f4f6fb",
				"muted":      "#a5adbb",
				"accent":     "#65d6ad",
				"border":     "rgba(255, 255, 255, 0.13)",
				"danger":     "#ff6b7a",
			},
		},
		AI: AISettings{
			Provider:     "local",
			Model:        "",
			CloudBaseURL: "https://api.openai.com/v1",
			Temperature:  0.2,
			MaxTokens:    2048,
		},
	}
}

func (s *Service) OpenWorkspace(root string) error {
	return s.workspace.Open(root)
}

func (s *Service) WorkspaceState() WorkspaceState {
	return s.workspace.State()
}

func (s *Service) Settings() Settings {
	return settingsForResponse(s.settingsSnapshot(false))
}

func (s *Service) UpdateSettings(settings Settings) Settings {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.settings = mergeSettings(s.settings, settings)
	_ = savePersistedSettings(s.settings)

	return settingsForResponse(s.settings)
}

func (s *Service) ListModels(ctx context.Context, providerID string) ([]string, error) {
	provider, err := s.providers.Get(providerID)
	if err != nil {
		return nil, err
	}
	if cloud, ok := provider.(*CloudProvider); ok {
		return cloud.ListModelsWithSettings(ctx, s.settingsSnapshot(false).AI)
	}
	return provider.ListModels(ctx)
}

func (s *Service) RunAgent(ctx context.Context, req AgentRunRequest) (*AgentRunResponse, error) {
	return s.runAgentLoop(ctx, req)
}

func (s *Service) Health() IDEHealth {
	return IDEHealth{
		Status:    "ok",
		Workspace: s.WorkspaceState(),
		Providers: []string{"local", "cloud"},
		Features: []string{
			"workspace",
			"file_tree",
			"tabs",
			"syntax_highlighting",
			"project_search",
			"settings",
			"themes",
			"local_models",
			"cloud_models",
			"agent_tools",
			"diff_review",
		},
		Limits: IDELimits{
			MaxFileSize:      s.workspace.maxFileSize,
			MaxTreeEntries:   maxTreeEntries,
			MaxSearchResults: maxSearchResults,
			MaxAgentRounds:   defaultAgentMaxRounds,
		},
		Settings: s.Settings(),
	}
}

func (s *Service) settingsSnapshot(maskSecret bool) Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()

	settings := s.settings
	if maskSecret {
		return settingsForResponse(settings)
	}
	return settings
}

func (s *Service) ApplyChanges(ctx context.Context, req ApplyChangesRequest) (*ApplyChangesResponse, error) {
	resp := &ApplyChangesResponse{}
	for _, change := range req.Changes {
		applied := change
		switch change.Action {
		case actionCreate:
			if err := s.workspace.CreateFile(ctx, change.Path, change.After); err != nil {
				applied.Error = err.Error()
				resp.Skipped = append(resp.Skipped, applied)
				continue
			}
			resp.Applied = append(resp.Applied, applied)
		case actionUpdate:
			current, err := s.workspace.ReadFile(ctx, change.Path)
			if err != nil {
				applied.Error = err.Error()
				resp.Skipped = append(resp.Skipped, applied)
				continue
			}
			if change.Before != "" && current.Content != change.Before {
				applied.Error = "file changed since diff was generated"
				resp.Skipped = append(resp.Skipped, applied)
				continue
			}
			if err := s.workspace.WriteFile(ctx, change.Path, change.After); err != nil {
				applied.Error = err.Error()
				resp.Skipped = append(resp.Skipped, applied)
				continue
			}
			resp.Applied = append(resp.Applied, applied)
		case actionDelete:
			if !req.ConfirmDelete {
				applied.Error = "delete requires confirmation"
				resp.Skipped = append(resp.Skipped, applied)
				continue
			}
			if err := s.workspace.DeleteFile(ctx, change.Path); err != nil {
				applied.Error = err.Error()
				resp.Skipped = append(resp.Skipped, applied)
				continue
			}
			resp.Applied = append(resp.Applied, applied)
		default:
			applied.Error = "unknown change action"
			resp.Skipped = append(resp.Skipped, applied)
		}
	}
	return resp, nil
}

func (s *Service) previewChange(ctx context.Context, change FileChange) FileChange {
	before := change.Before
	if before == "" && change.Action != actionCreate {
		if current, err := s.workspace.ReadFile(ctx, change.Path); err == nil {
			before = current.Content
		}
	}

	action := strings.ToLower(strings.TrimSpace(change.Action))
	if action == "" {
		action = actionUpdate
	}
	if action == actionDelete {
		return BuildChange(change.Path, actionDelete, before, "")
	}
	if action == actionCreate {
		return BuildChange(change.Path, actionCreate, "", change.After)
	}
	return BuildChange(change.Path, actionUpdate, before, change.After)
}

func (s *Service) collectAgentFiles(ctx context.Context, requested []string) ([]FileContent, []ToolResult) {
	paths := requested
	if len(paths) == 0 {
		paths = defaultContextPaths(s.workspace.State().Root)
	}

	files := make([]FileContent, 0, len(paths))
	results := make([]ToolResult, 0, len(paths)+1)
	total := 0
	for _, path := range paths {
		if total >= maxAgentContextChars {
			break
		}
		file, err := s.workspace.ReadFile(ctx, path)
		if err != nil {
			results = append(results, ToolResult{Tool: "read_file", Path: path, Error: err.Error()})
			continue
		}
		content := file.Content
		remaining := maxAgentContextChars - total
		if len(content) > remaining {
			content = content[:remaining]
		}
		total += len(content)
		files = append(files, FileContent{
			Path:     file.Path,
			Content:  content,
			Language: file.Language,
			Encoding: file.Encoding,
			Size:     file.Size,
		})
		results = append(results, ToolResult{Tool: "read_file", Path: file.Path, Output: map[string]any{"bytes": len(content)}})
	}
	return files, results
}

func defaultContextPaths(root string) []string {
	candidates := []string{"README.md", "go.mod", "main.go", "package.json", "src/main.ts", "src/main.js"}
	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(candidate))); err == nil {
			paths = append(paths, candidate)
		}
	}
	return paths
}

func buildAgentPrompt(task string, state WorkspaceState, tree []FileNode, files []FileContent) (string, error) {
	treeJSON, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Workspace: %s\nRoot name: %s\n\n", state.Root, state.Name)
	fmt.Fprintf(&b, "User task:\n%s\n\n", task)
	b.WriteString("Project tree sample:\n")
	b.Write(treeJSON)
	b.WriteString("\n\nFiles available to you:\n")
	for _, file := range files {
		fmt.Fprintf(&b, "\n--- FILE: %s (%s) ---\n%s\n", file.Path, file.Language, file.Content)
	}
	b.WriteString("\nReturn strict JSON only. Do not wrap it in markdown.\n")
	return b.String(), nil
}

func agentSystemPrompt() string {
	return `You are an AI coding agent inside a local IDE.
You must preserve existing backend behavior, public APIs, auth flows, database contracts, and current data structures unless the user explicitly asks otherwise.
Work through safe tools conceptually: read_file, write_file, create_file, delete_file only with confirmation, list_directory, search_project, apply_patch.
Return a JSON object with this shape:
{
  "message": "short explanation",
  "changes": [
    {"path": "relative/path.ext", "action": "create|update|delete", "after": "full new file content for create/update"}
  ]
}
For updates, always provide the full final file content in "after".
Prefer additive modules and small route wiring over rewrites. If no code change is needed, return an empty changes array.`
}

type agentJSONResponse struct {
	Message string `json:"message"`
	Changes []agentChangeProposal `json:"changes"`
}

type agentChangeProposal struct {
	Path    string `json:"path"`
	Action  string `json:"action"`
	After   string `json:"after"`
	Content string `json:"content"`
}

func parseAgentJSON(text string) (*agentJSONResponse, error) {
	clean := extractJSONObject(text)

	var parsed agentJSONResponse
	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (s *Service) normalizeProposal(ctx context.Context, proposal agentChangeProposal) FileChange {
	path := filepath.ToSlash(strings.TrimSpace(proposal.Path))
	action := strings.ToLower(strings.TrimSpace(proposal.Action))
	after := proposal.After
	if after == "" {
		after = proposal.Content
	}

	switch action {
	case actionCreate:
		return BuildChange(path, actionCreate, "", after)
	case actionDelete:
		before := ""
		if current, err := s.workspace.ReadFile(ctx, path); err == nil {
			before = current.Content
		}
		return BuildChange(path, actionDelete, before, "")
	case actionUpdate:
		fallthrough
	default:
		before := ""
		changeAction := actionUpdate
		if current, err := s.workspace.ReadFile(ctx, path); err == nil {
			before = current.Content
		} else {
			changeAction = actionCreate
		}
		return BuildChange(path, changeAction, before, after)
	}
}
