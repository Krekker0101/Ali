package ide

import (
	"context"
	"time"
)

const (
	actionCreate = "create"
	actionUpdate = "update"
	actionDelete = "delete"
)

type WorkspaceState struct {
	Root     string    `json:"root"`
	Name     string    `json:"name"`
	OpenedAt time.Time `json:"opened_at"`
}

type FileNode struct {
	Path     string     `json:"path"`
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	Size     int64      `json:"size"`
	Modified time.Time  `json:"modified"`
	Language string     `json:"language"`
	Children []FileNode `json:"children,omitempty"`
}

type FileContent struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
	Encoding string `json:"encoding"`
	Size     int64  `json:"size"`
}

type SearchResult struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Preview string `json:"preview"`
}

type FileChange struct {
	Path                 string `json:"path"`
	Action               string `json:"action"`
	Before               string `json:"before,omitempty"`
	After                string `json:"after,omitempty"`
	Diff                 string `json:"diff"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
	Error                string `json:"error,omitempty"`
}

type ApplyChangesRequest struct {
	Changes       []FileChange `json:"changes"`
	ConfirmDelete bool         `json:"confirm_delete"`
}

type ApplyChangesResponse struct {
	Applied []FileChange `json:"applied"`
	Skipped []FileChange `json:"skipped"`
}

type ThemeSettings struct {
	Mode   string            `json:"mode"`
	Colors map[string]string `json:"colors"`
}

type AISettings struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	CloudBaseURL string  `json:"cloud_base_url"`
	CloudAPIKey  string  `json:"cloud_api_key,omitempty"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
}

type Settings struct {
	Theme ThemeSettings `json:"theme"`
	AI    AISettings    `json:"ai"`
}

type IDEHealth struct {
	Status    string         `json:"status"`
	Workspace WorkspaceState `json:"workspace"`
	Providers []string       `json:"providers"`
	Features  []string       `json:"features"`
	Limits    IDELimits      `json:"limits"`
	Settings  Settings       `json:"settings"`
}

type IDELimits struct {
	MaxFileSize      int64 `json:"max_file_size"`
	MaxTreeEntries   int   `json:"max_tree_entries"`
	MaxSearchResults int   `json:"max_search_results"`
	MaxAgentRounds   int   `json:"max_agent_rounds"`
}

type CompletionRequest struct {
	Model       string
	System      string
	Prompt      string
	Temperature float64
	MaxTokens   int
}

type CompletionResponse struct {
	Text     string `json:"text"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type Provider interface {
	ID() string
	ListModels(ctx context.Context) ([]string, error)
	Complete(ctx context.Context, req CompletionRequest, settings AISettings) (CompletionResponse, error)
}

type AgentRunRequest struct {
	Task     string   `json:"task"`
	Files    []string `json:"files"`
	Provider string   `json:"provider"`
	Model    string   `json:"model"`
}

type AgentToolCall struct {
	Tool      string         `json:"tool"`
	Name      string         `json:"name,omitempty"`
	Args      map[string]any `json:"args,omitempty"`
	Arguments any            `json:"arguments,omitempty"`
}

type ToolResult struct {
	Tool   string `json:"tool"`
	Path   string `json:"path,omitempty"`
	Output any    `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

type AgentStep struct {
	Round     int             `json:"round"`
	Thought   string          `json:"thought,omitempty"`
	ToolCalls []AgentToolCall `json:"tool_calls,omitempty"`
	Results   []ToolResult    `json:"results,omitempty"`
}

type AgentRunResponse struct {
	Message     string       `json:"message"`
	Provider    string       `json:"provider"`
	Model       string       `json:"model"`
	Changes     []FileChange `json:"changes"`
	Steps       []AgentStep  `json:"steps,omitempty"`
	ToolResults []ToolResult `json:"tool_results"`
	Raw         string       `json:"raw,omitempty"`
}
