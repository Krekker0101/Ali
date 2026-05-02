package ide

import (
	"strings"
	"testing"
)

func TestUnifiedDiffForCreate(t *testing.T) {
	diff := UnifiedDiff("new.txt", actionCreate, "", "hello\n")
	if !strings.Contains(diff, "--- /dev/null") || !strings.Contains(diff, "+hello") {
		t.Fatalf("unexpected diff:\n%s", diff)
	}
}

func TestBuildChangeMarksDeleteForConfirmation(t *testing.T) {
	change := BuildChange("old.txt", actionDelete, "gone", "")
	if !change.RequiresConfirmation {
		t.Fatal("delete change should require confirmation")
	}
}
