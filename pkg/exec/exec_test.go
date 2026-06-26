package exec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteRunsCommand(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.txt")

	// Use `sh -c` to write the path to a file so we can verify substitution.
	cs, err := NewCommandSet([][]string{{"sh", "-c", "printf '%s\\n' \"$1\" >> " + out, "sh"}})
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
	if got := string(data); got != "hello/world.go\n" {
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
	cs, err := NewCommandSet([][]string{{"sh", "-c", "printf '%s' \"$1\" > " + out, "sh"}})
	if err != nil {
		t.Fatal(err)
	}
	// No explicit placeholder -> appended at end as an argument.
	cs.Execute("abc", "", true)
	data, _ := os.ReadFile(out)
	if string(data) != "abc" {
		t.Errorf("got %q", string(data))
	}
}
