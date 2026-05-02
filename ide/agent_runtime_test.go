package ide

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseAgentLoopReplyAcceptsToolCall(t *testing.T) {
	reply, err := parseAgentLoopReply(`{"thought":"need file","tool_calls":[{"name":"read_file","arguments":"{\"path\":\"README.md\"}"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.ToolCalls) != 1 || reply.ToolCalls[0].Name != "read_file" {
		t.Fatalf("unexpected reply: %#v", reply)
	}
}

func TestAgentPrepareChangeDoesNotWriteFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	service, err := NewService(root)
	if err != nil {
		t.Fatal(err)
	}

	loop := &agentLoopState{}
	result := service.executeAgentTool(context.Background(), AgentToolCall{
		Tool: "write_file",
		Args: map[string]any{
			"path":    "main.go",
			"content": "package main\n\nfunc main() {}\n",
		},
	}, loop)
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	if len(loop.pendingChanges) != 1 {
		t.Fatalf("pending changes = %d, want 1", len(loop.pendingChanges))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "package main\n" {
		t.Fatal("agent tool wrote to disk before approval")
	}
}

func TestAgentToolAcceptsJSONStringArguments(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	service, err := NewService(root)
	if err != nil {
		t.Fatal(err)
	}

	loop := &agentLoopState{}
	result := service.executeAgentTool(context.Background(), AgentToolCall{
		Name:      "read_file",
		Arguments: `{"path":"README.md"}`,
	}, loop)
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	if result.Path != "README.md" {
		t.Fatalf("path = %q, want README.md", result.Path)
	}
}
