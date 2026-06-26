package exec

import (
	"os"
	"path/filepath"
	"testing"
)

const execHelperMarker = "gofd-exec-helper"

func TestExecuteRunsCommand(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.txt")

	cs, err := NewCommandSet([][]string{testHelperCommand(out, "{}")})
	if err != nil {
		t.Fatal(err)
	}
	if !cs.Execute("hello/world.go", "", true) {
		t.Fatal("Execute reported failure")
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "hello/world.go" {
		t.Errorf("got %q", got)
	}
}

func TestBatchValidation(t *testing.T) {
	if _, err := NewBatchCommandSet([][]string{{"echo", "{.}", "{}"}}); err == nil {
		t.Error("expected error for multiple placeholders in batch mode")
	}
	if _, err := NewCommandSet([][]string{{}}); err == nil {
		t.Error("expected error for empty command")
	}
}

func TestImplicitPlaceholder(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.txt")
	cs, err := NewCommandSet([][]string{testHelperCommand(out)})
	if err != nil {
		t.Fatal(err)
	}
	// No explicit placeholder -> appended at end as an argument.
	if !cs.Execute("abc", "", true) {
		t.Fatal("Execute reported failure")
	}
	data, _ := os.ReadFile(out)
	if string(data) != "abc" {
		t.Errorf("got %q", string(data))
	}
}

func testHelperCommand(out string, args ...string) []string {
	cmd := []string{os.Args[0], "-test.run=^TestExecHelperProcess$", execHelperMarker, out}
	return append(cmd, args...)
}

func TestExecHelperProcess(t *testing.T) {
	marker := -1
	for i, arg := range os.Args {
		if arg == execHelperMarker {
			marker = i
			break
		}
	}
	if marker == -1 {
		return
	}
	if marker+1 >= len(os.Args) {
		t.Fatal("missing output path")
	}
	payload := ""
	if marker+2 < len(os.Args) {
		payload = os.Args[marker+2]
	}
	if err := os.WriteFile(os.Args[marker+1], []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
}
