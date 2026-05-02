package ide

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	w, err := NewWorkspace(root, 1024)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.ReadFile(context.Background(), "../outside.txt"); err == nil {
		t.Fatal("expected traversal read to fail")
	}
	if err := w.WriteFile(context.Background(), "../outside.txt", "bad"); err == nil {
		t.Fatal("expected traversal write to fail")
	}
}

func TestWorkspaceReadsAndSearchesFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWorkspace(root, 1024)
	if err != nil {
		t.Fatal(err)
	}

	file, err := w.ReadFile(context.Background(), "src/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if file.Language != "go" {
		t.Fatalf("language = %q, want go", file.Language)
	}

	results, err := w.Search(context.Background(), "func", ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Path != "src/main.go" {
		t.Fatalf("unexpected results: %#v", results)
	}
}
