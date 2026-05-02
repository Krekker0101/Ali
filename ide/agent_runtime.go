package ide

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultAgentMaxRounds = 8
	maxToolOutputBytes    = 14000
)

type agentLoopReply struct {
	Thought   string                `json:"thought"`
	Done      bool                  `json:"done"`
	Message   string                `json:"message"`
	ToolCalls []AgentToolCall       `json:"tool_calls"`
	Changes   []agentChangeProposal `json:"changes"`
}

type agentLoopState struct {
	task           string
	settings       Settings
	provider       Provider
	selectedFiles  []string
	pendingChanges []FileChange
	steps          []AgentStep
	history        []agentHistoryItem
	toolResults    []ToolResult
}

type agentHistoryItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (s *Service) runAgentLoop(ctx context.Context, req AgentRunRequest) (*AgentRunResponse, error) {
	task := strings.TrimSpace(req.Task)
	if task == "" {
		return nil, fmt.Errorf("task is required")
	}

	settings := s.settingsSnapshot(false)
	if req.Provider != "" {
		settings.AI.Provider = req.Provider
	}
	if req.Model != "" {
		settings.AI.Model = req.Model
	}

	provider, err := s.providers.Get(settings.AI.Provider)
	if err != nil {
		return nil, err
	}

	loop := &agentLoopState{
		task:          task,
		settings:      settings,
		provider:      provider,
		selectedFiles: req.Files,
	}

	seedResults, err := s.seedAgentContext(ctx, loop)
	if err != nil {
		return nil, err
	}
	loop.toolResults = append(loop.toolResults, seedResults...)

	for round := 1; round <= defaultAgentMaxRounds; round++ {
		prompt, err := s.buildLoopPrompt(ctx, loop, round)
		if err != nil {
			return nil, err
		}

		completion, err := provider.Complete(ctx, CompletionRequest{
			Model:       settings.AI.Model,
			System:      agentLoopSystemPrompt(),
			Prompt:      prompt,
			Temperature: settings.AI.Temperature,
			MaxTokens:   settings.AI.MaxTokens,
		}, settings.AI)
		if err != nil {
			return nil, err
		}

		reply, err := parseAgentLoopReply(completion.Text)
		if err != nil {
			return &AgentRunResponse{
				Message:     completion.Text,
				Provider:    completion.Provider,
				Model:       completion.Model,
				Changes:     loop.pendingChanges,
				Steps:       loop.steps,
				ToolResults: loop.toolResults,
				Raw:         completion.Text,
			}, nil
		}

		step := AgentStep{
			Round:     round,
			Thought:   reply.Thought,
			ToolCalls: reply.ToolCalls,
		}

		for _, proposal := range reply.Changes {
			change := s.normalizeProposal(ctx, proposal)
			loop.pendingChanges = upsertChange(loop.pendingChanges, change)
		}

		if reply.Done || len(reply.ToolCalls) == 0 {
			message := strings.TrimSpace(reply.Message)
			if message == "" {
				message = "Agent completed the task and prepared changes for review."
			}
			loop.steps = append(loop.steps, step)
			return &AgentRunResponse{
				Message:     message,
				Provider:    completion.Provider,
				Model:       completion.Model,
				Changes:     loop.pendingChanges,
				Steps:       loop.steps,
				ToolResults: loop.toolResults,
				Raw:         completion.Text,
			}, nil
		}

		for _, call := range reply.ToolCalls {
			result := s.executeAgentTool(ctx, call, loop)
			step.Results = append(step.Results, result)
			loop.toolResults = append(loop.toolResults, result)
		}

		loop.steps = append(loop.steps, step)
		loop.history = append(loop.history, agentHistoryItem{
			Role:    "assistant",
			Content: trimForPrompt(completion.Text, maxToolOutputBytes),
		})
		resultsJSON, _ := json.Marshal(step.Results)
		loop.history = append(loop.history, agentHistoryItem{
			Role:    "tool",
			Content: trimForPrompt(string(resultsJSON), maxToolOutputBytes),
		})
	}

	return &AgentRunResponse{
		Message:     "Agent reached the step limit. Review the prepared changes or narrow the task and run again.",
		Provider:    settings.AI.Provider,
		Model:       settings.AI.Model,
		Changes:     loop.pendingChanges,
		Steps:       loop.steps,
		ToolResults: loop.toolResults,
	}, nil
}

func (s *Service) seedAgentContext(ctx context.Context, loop *agentLoopState) ([]ToolResult, error) {
	paths := loop.selectedFiles
	if len(paths) == 0 {
		paths = defaultContextPaths(s.workspace.State().Root)
	}
	loop.selectedFiles = paths

	results := make([]ToolResult, 0, len(paths)+1)
	tree, err := s.workspace.Tree(ctx, ".", 2)
	if err == nil {
		results = append(results, ToolResult{Tool: "project_tree", Path: ".", Output: tree})
	}

	for _, path := range paths {
		file, err := s.workspace.ReadFile(ctx, path)
		if err != nil {
			results = append(results, ToolResult{Tool: "read_file", Path: path, Error: err.Error()})
			continue
		}
		results = append(results, ToolResult{
			Tool: "read_file",
			Path: file.Path,
			Output: map[string]any{
				"language": file.Language,
				"size":     file.Size,
				"content":  trimForPrompt(file.Content, maxAgentContextChars),
			},
		})
	}
	return results, nil
}

func (s *Service) buildLoopPrompt(ctx context.Context, loop *agentLoopState, round int) (string, error) {
	state := s.WorkspaceState()
	seedJSON, err := json.MarshalIndent(loop.toolResults, "", "  ")
	if err != nil {
		return "", err
	}
	historyJSON, err := json.MarshalIndent(loop.history, "", "  ")
	if err != nil {
		return "", err
	}
	pendingJSON, err := json.MarshalIndent(loop.pendingChanges, "", "  ")
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Round: %d of %d\n", round, defaultAgentMaxRounds)
	fmt.Fprintf(&b, "Workspace root: %s\nWorkspace name: %s\n", state.Root, state.Name)
	fmt.Fprintf(&b, "Selected files: %s\n\n", strings.Join(loop.selectedFiles, ", "))
	fmt.Fprintf(&b, "User task:\n%s\n\n", loop.task)
	b.WriteString("Known tool results and project context:\n")
	b.Write(seedJSON)
	b.WriteString("\n\nConversation history:\n")
	b.Write(historyJSON)
	b.WriteString("\n\nPending diff proposals:\n")
	b.Write(pendingJSON)
	b.WriteString("\n\nRespond with strict JSON only.\n")
	return b.String(), nil
}

func agentLoopSystemPrompt() string {
	return `You are a professional autonomous coding agent embedded in an IDE.
Your goal is Cursor/Codex style behavior: inspect the project, reason over files, use tools, prepare precise code changes, and preserve existing backend behavior.

Hard safety contract:
- Do not change existing public APIs, auth, database contracts, or backend behavior unless the user explicitly asks.
- Prefer additive modules, narrow edits, and compatibility.
- Never claim a file was changed. You only prepare diffs. The user applies them later.
- Delete operations must be proposed as delete changes and require user confirmation.
- For update/create changes you must provide the full final file content in "after".

Available tools. Return them in tool_calls:
- list_directory: {"path":"relative/path"}
- project_tree: {"path":"relative/path","depth":1-5}
- read_file: {"path":"relative/file"}
- search_project: {"query":"text","path":"relative/path"}
- prepare_change: {"path":"relative/file","action":"create|update|delete","after":"full content for create/update"}
- write_file and create_file are aliases for prepare_change and do not write to disk.
- delete_file only prepares a delete proposal.
- apply_patch: {"changes":[{"path":"relative/file","action":"create|update|delete","after":"full content"}]}

Response schema for another step:
{"thought":"what you need next","tool_calls":[{"tool":"read_file","args":{"path":"README.md"}}]}

Response schema for final answer:
{"done":true,"message":"short summary for the user","changes":[{"path":"relative/file","action":"update","after":"full final content"}]}

Use tools when context is insufficient. Finish only when the proposed changes are coherent and reviewable.`
}

func parseAgentLoopReply(text string) (*agentLoopReply, error) {
	clean := extractJSONObject(text)
	var reply agentLoopReply
	if err := json.Unmarshal([]byte(clean), &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

func (s *Service) executeAgentTool(ctx context.Context, call AgentToolCall, loop *agentLoopState) ToolResult {
	tool := strings.ToLower(strings.TrimSpace(call.Tool))
	if tool == "" {
		tool = strings.ToLower(strings.TrimSpace(call.Name))
	}
	args := call.Args
	if args == nil {
		args = mapArg(call.Arguments)
	}
	path := stringArg(args, "path")
	if path == "" {
		path = "."
	}

	switch tool {
	case "list_directory":
		nodes, err := s.workspace.ListDir(ctx, path)
		return toolResult(tool, path, nodes, err)
	case "project_tree":
		depth := intArg(args, "depth", 3)
		nodes, err := s.workspace.Tree(ctx, path, depth)
		return toolResult(tool, path, nodes, err)
	case "read_file":
		file, err := s.workspace.ReadFile(ctx, path)
		if err != nil {
			return toolResult(tool, path, nil, err)
		}
		output := map[string]any{
			"path":     file.Path,
			"language": file.Language,
			"size":     file.Size,
			"content":  trimForPrompt(file.Content, maxToolOutputBytes),
		}
		return toolResult(tool, path, output, nil)
	case "search_project":
		query := stringArg(args, "query")
		results, err := s.workspace.Search(ctx, query, path)
		return toolResult(tool, path, results, err)
	case "prepare_change", "write_file", "create_file", "delete_file":
		change := s.changeFromTool(ctx, tool, args)
		loop.pendingChanges = upsertChange(loop.pendingChanges, change)
		return ToolResult{
			Tool: tool,
			Path: change.Path,
			Output: map[string]any{
				"action":                change.Action,
				"diff":                  change.Diff,
				"requires_confirmation": change.RequiresConfirmation,
			},
			Error: change.Error,
		}
	case "apply_patch":
		changes, err := s.changesFromPatchTool(ctx, args)
		if err != nil {
			return toolResult(tool, path, nil, err)
		}
		for _, change := range changes {
			loop.pendingChanges = upsertChange(loop.pendingChanges, change)
		}
		return toolResult(tool, path, map[string]any{"changes": changes}, nil)
	default:
		return ToolResult{Tool: tool, Path: path, Error: "unknown tool"}
	}
}

func (s *Service) changeFromTool(ctx context.Context, tool string, args map[string]any) FileChange {
	path := filepath.ToSlash(strings.TrimSpace(stringArg(args, "path")))
	action := strings.ToLower(strings.TrimSpace(stringArg(args, "action")))
	after := stringArg(args, "after")
	if _, ok := args["after"]; !ok {
		after = stringArg(args, "content")
	}

	if tool == "create_file" {
		action = actionCreate
	}
	if tool == "write_file" {
		action = actionUpdate
	}
	if tool == "delete_file" {
		action = actionDelete
	}
	if action == "" {
		action = actionUpdate
	}

	change := s.normalizeProposal(ctx, agentChangeProposal{
		Path:   path,
		Action: action,
		After:  after,
	})
	if path == "" {
		change.Error = "path is required"
	}
	if (change.Action == actionCreate || change.Action == actionUpdate) && after == "" {
		change.Error = "after content is required for create/update"
	}
	return change
}

func (s *Service) changesFromPatchTool(ctx context.Context, args map[string]any) ([]FileChange, error) {
	raw, ok := args["changes"]
	if !ok {
		return nil, fmt.Errorf("apply_patch requires a changes array")
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	var proposals []agentChangeProposal
	if err := json.Unmarshal(data, &proposals); err != nil {
		return nil, err
	}

	changes := make([]FileChange, 0, len(proposals))
	for _, proposal := range proposals {
		changes = append(changes, s.normalizeProposal(ctx, proposal))
	}
	return changes, nil
}

func toolResult(tool string, path string, output any, err error) ToolResult {
	result := ToolResult{Tool: tool, Path: path, Output: output}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

func upsertChange(changes []FileChange, next FileChange) []FileChange {
	for i, existing := range changes {
		if existing.Path == next.Path && existing.Action == next.Action {
			changes[i] = next
			return changes
		}
	}
	return append(changes, next)
}

func extractJSONObject(text string) string {
	clean := strings.TrimSpace(text)
	if strings.HasPrefix(clean, "```") {
		clean = strings.TrimPrefix(clean, "```json")
		clean = strings.TrimPrefix(clean, "```")
		clean = strings.TrimSuffix(clean, "```")
		clean = strings.TrimSpace(clean)
	}
	if strings.HasPrefix(clean, "{") {
		return clean
	}
	start := strings.Index(clean, "{")
	end := strings.LastIndex(clean, "}")
	if start >= 0 && end > start {
		return clean[start : end+1]
	}
	return clean
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func mapArg(value any) map[string]any {
	switch v := value.(type) {
	case nil:
		return nil
	case map[string]any:
		return v
	case string:
		var out map[string]any
		if err := json.Unmarshal([]byte(v), &out); err == nil {
			return out
		}
	}
	return nil
}

func intArg(args map[string]any, key string, fallback int) int {
	raw := stringArg(args, key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func trimForPrompt(content string, limit int) string {
	if limit <= 0 || len(content) <= limit {
		return content
	}
	return content[:limit] + "\n... truncated ..."
}
