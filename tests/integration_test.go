package tests

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	gofd "github.com/startvibecoding/go-fd"
)

// setupTree builds a small test directory tree and returns its path.
func setupTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite := func(rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("src/main.go", "package main")
	mustWrite("src/util.go", "package main")
	mustWrite("docs/README.md", "# docs")
	mustWrite("node_modules/lib.js", "x")
	mustWrite(".hidden", "secret")
	mustWrite("Makefile", "all:")
	mustWrite(".gitignore", "node_modules\n*.md\n")
	return root
}

func relSorted(t *testing.T, root string, paths []string) []string {
	t.Helper()
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		r, err := filepath.Rel(root, p)
		if err != nil {
			r = p
		}
		out = append(out, filepath.ToSlash(r))
	}
	sort.Strings(out)
	return out
}

func TestSDKBasicSearch(t *testing.T) {
	root := setupTree(t)
	got, err := gofd.Find(context.Background(), gofd.Options{
		Pattern: `\.go$`,
		Paths:   []string{root},
	})
	if err != nil {
		t.Fatal(err)
	}
	rel := relSorted(t, root, got)
	want := []string{"src/main.go", "src/util.go"}
	if !equal(rel, want) {
		t.Errorf("got %v, want %v", rel, want)
	}
}

func TestSDKRespectsGitignore(t *testing.T) {
	root := setupTree(t)
	got, _ := gofd.Find(context.Background(), gofd.Options{Paths: []string{root}})
	for _, p := range got {
		if filepath.Base(p) == "lib.js" {
			t.Errorf("node_modules should be ignored: %s", p)
		}
		if filepath.Ext(p) == ".md" {
			t.Errorf(".md files should be ignored: %s", p)
		}
	}
}

func TestSDKUnrestricted(t *testing.T) {
	root := setupTree(t)
	got, _ := gofd.Find(context.Background(), gofd.Options{
		Paths:        []string{root},
		Unrestricted: true,
	})
	found := map[string]bool{}
	for _, p := range got {
		found[filepath.Base(p)] = true
	}
	for _, name := range []string{".hidden", "README.md", "lib.js"} {
		if !found[name] {
			t.Errorf("unrestricted search should include %s", name)
		}
	}
}

func TestSDKExtensionFilter(t *testing.T) {
	root := setupTree(t)
	got, _ := gofd.Find(context.Background(), gofd.Options{
		Paths:      []string{root},
		Extensions: []string{"go"},
	})
	if len(got) != 2 {
		t.Errorf("expected 2 .go files, got %d (%v)", len(got), got)
	}
}

func TestSDKTypeFilter(t *testing.T) {
	root := setupTree(t)
	got, _ := gofd.Find(context.Background(), gofd.Options{
		Paths: []string{root},
		Types: []string{"d"},
	})
	rel := relSorted(t, root, got)
	want := []string{"docs", "src"}
	if !equal(rel, want) {
		t.Errorf("got %v, want %v", rel, want)
	}
}

func TestSDKGlob(t *testing.T) {
	root := setupTree(t)
	got, _ := gofd.Find(context.Background(), gofd.Options{
		Pattern: "*.go",
		Glob:    true,
		Paths:   []string{root},
	})
	if len(got) != 2 {
		t.Errorf("expected 2, got %v", got)
	}
}

func TestSDKMaxDepth(t *testing.T) {
	root := setupTree(t)
	got, _ := gofd.Find(context.Background(), gofd.Options{
		Paths:    []string{root},
		MaxDepth: 1,
	})
	for _, p := range got {
		rel, _ := filepath.Rel(root, p)
		if len(filepath.SplitList(rel)) > 1 {
			continue
		}
	}
	// Depth 1 should not include src/main.go.
	for _, p := range got {
		if filepath.Base(p) == "main.go" {
			t.Errorf("max-depth 1 should not include %s", p)
		}
	}
}

func TestSDKStream(t *testing.T) {
	root := setupTree(t)
	results, errs, err := gofd.Stream(context.Background(), gofd.Options{
		Pattern: `\.go$`,
		Paths:   []string{root},
	})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for range results {
		count++
	}
	for range errs {
	}
	if count != 2 {
		t.Errorf("expected 2 streamed results, got %d", count)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
