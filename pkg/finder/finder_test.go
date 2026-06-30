package finder

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

func TestMatchesPattern(t *testing.T) {
	f, err := New(&Config{})
	if err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(`(?is)\.go$`)
	f.SetPatterns([]*regexp.Regexp{re})

	// Matching entry.
	entry := &DirEntry{path: "src/main.go", depth: 2}
	if !f.matches(entry) {
		t.Error("expected main.go to match")
	}

	// Non-matching entry.
	entry2 := &DirEntry{path: "src/main.rs", depth: 2}
	if f.matches(entry2) {
		t.Error("expected main.rs to not match")
	}

	// Empty pattern matches everything.
	f2, _ := New(&Config{})
	f2.SetPatterns([]*regexp.Regexp{})
	if !f2.matches(entry2) {
		t.Error("empty pattern should match everything")
	}
}

func TestMatchesMultiplePatterns(t *testing.T) {
	f, err := New(&Config{})
	if err != nil {
		t.Fatal(err)
	}

	// Both patterns must match (AND semantics).
	re1 := regexp.MustCompile(`(?is)main`)
	re2 := regexp.MustCompile(`(?is)\.go$`)
	f.SetPatterns([]*regexp.Regexp{re1, re2})

	entry := &DirEntry{path: "src/main.go", depth: 2}
	if !f.matches(entry) {
		t.Error("expected main.go to match both patterns")
	}

	entry2 := &DirEntry{path: "src/util.go", depth: 2}
	if f.matches(entry2) {
		t.Error("util.go should not match 'main' pattern")
	}
}

func TestMatchesCaseSensitive(t *testing.T) {
	f, err := New(&Config{CaseSensitive: true})
	if err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(`(?s)README`)
	f.SetPatterns([]*regexp.Regexp{re})

	entry := &DirEntry{path: "README.md", depth: 1}
	if !f.matches(entry) {
		t.Error("expected README.md to match (case sensitive)")
	}

	entry2 := &DirEntry{path: "readme.md", depth: 1}
	if f.matches(entry2) {
		t.Error("readme.md should not match case-sensitive 'README'")
	}
}

func TestMatchesCaseInsensitive(t *testing.T) {
	f, err := New(&Config{CaseSensitive: false})
	if err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(`(?is)readme`)
	f.SetPatterns([]*regexp.Regexp{re})

	entry := &DirEntry{path: "README.md", depth: 1}
	if !f.matches(entry) {
		t.Error("expected README.md to match (case insensitive)")
	}
}

func TestMatchesFullPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("cannot determine cwd")
	}

	f, err := New(&Config{FullPathBase: cwd})
	if err != nil {
		t.Fatal(err)
	}

	// Pattern matches the full path. Use a unique prefix unlikely in cwd.
	re := regexp.MustCompile(`(?is)gofd_finder_test/.*\.go$`)
	f.SetPatterns([]*regexp.Regexp{re})

	entry := &DirEntry{path: "gofd_finder_test/main.go", depth: 2}
	if !f.matches(entry) {
		t.Error("expected gofd_finder_test/main.go to match full-path pattern")
	}

	entry2 := &DirEntry{path: "other_test/main.go", depth: 2}
	if f.matches(entry2) {
		t.Error("other_test/main.go should not match gofd_finder_test/.* pattern")
	}
}

func TestMatchesFullPathUsesSlashSeparators(t *testing.T) {
	f, err := New(&Config{FullPathBase: `C:\work\go-fd`})
	if err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(`(?is)go-fd/gofd_finder_test/.*\.go$`)
	f.SetPatterns([]*regexp.Regexp{re})

	entry := &DirEntry{path: `gofd_finder_test\main.go`, depth: 2}
	if !f.matches(entry) {
		t.Error("expected full-path pattern with slash separators to match Windows-style path")
	}
}

func TestMatchesMinDepth(t *testing.T) {
	depth := 2
	f, err := New(&Config{MinDepth: &depth})
	if err != nil {
		t.Fatal(err)
	}
	f.SetPatterns([]*regexp.Regexp{})

	shallow := &DirEntry{path: "main.go", depth: 1}
	if f.matches(shallow) {
		t.Error("depth 1 should not match min-depth 2")
	}

	deep := &DirEntry{path: "src/main.go", depth: 2}
	if !f.matches(deep) {
		t.Error("depth 2 should match min-depth 2")
	}
}

func TestStrippedPath(t *testing.T) {
	cfg := &Config{StripCwdPrefix: true}
	entry := &DirEntry{path: "./src/main.go", depth: 2}

	got := entry.StrippedPath(cfg)
	if got != "src/main.go" {
		t.Errorf("expected 'src/main.go', got %q", got)
	}

	cfg2 := &Config{StripCwdPrefix: false}
	got2 := entry.StrippedPath(cfg2)
	if got2 != "./src/main.go" {
		t.Errorf("expected './src/main.go', got %q", got2)
	}
}

func TestNewFinderDefaultThreads(t *testing.T) {
	cfg := &Config{Threads: 0}
	f, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Threads < 1 {
		t.Error("expected threads to be set to at least 1")
	}
	_ = f // avoid unused
}

func TestFindBasic(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.go", "b.go", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	f, err := New(&Config{
		IgnoreHidden:     true,
		ReadFdignore:     false,
		ReadVcsignore:    false,
		RequireGit:       true,
		ReadParentIgnore: false,
		ReadGlobalIgnore: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(`(?is)\.go$`)
	f.SetPatterns([]*regexp.Regexp{re})

	results, err := f.Find(context.Background(), []string{dir})
	if err != nil {
		t.Fatal(err)
	}

	sort.Strings(results)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(results), results)
	}
	// Results are relative to the search root, so they'll be just filenames.
	for _, r := range results {
		base := filepath.Base(r)
		if base != "a.go" && base != "b.go" {
			t.Errorf("unexpected result: %s", r)
		}
	}
}

func TestFindMaxResults(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.go", "b.go", "c.go"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	limit := 2
	f, err := New(&Config{
		MaxResults:       &limit,
		IgnoreHidden:     true,
		ReadFdignore:     false,
		ReadVcsignore:    false,
		RequireGit:       true,
		ReadParentIgnore: false,
		ReadGlobalIgnore: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	f.SetPatterns([]*regexp.Regexp{regexp.MustCompile(`(?is)\.go$`)})

	results, err := f.Find(context.Background(), []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(results))
	}
}

func TestFindHiddenIgnored(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".hidden", "visible.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	f, err := New(&Config{
		IgnoreHidden:     true,
		ReadFdignore:     false,
		ReadVcsignore:    false,
		RequireGit:       true,
		ReadParentIgnore: false,
		ReadGlobalIgnore: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	f.SetPatterns([]*regexp.Regexp{regexp.MustCompile(`(?is).*`)})

	results, err := f.Find(context.Background(), []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if filepath.Base(r) == ".hidden" {
			t.Error(".hidden should be ignored")
		}
	}
}

func TestFindHiddenIncluded(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".hidden", "visible.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	f, err := New(&Config{
		IgnoreHidden:     false,
		ReadFdignore:     false,
		ReadVcsignore:    false,
		RequireGit:       true,
		ReadParentIgnore: false,
		ReadGlobalIgnore: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	f.SetPatterns([]*regexp.Regexp{regexp.MustCompile(`(?is)hidden`)})

	results, err := f.Find(context.Background(), []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range results {
		if filepath.Base(r) == ".hidden" {
			found = true
		}
	}
	if !found {
		t.Error(".hidden should be found when hidden is included")
	}
}

func TestFinderConfig(t *testing.T) {
	cfg := &Config{Threads: 4}
	f, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if f.Config() != cfg {
		t.Error("Config() should return the same config pointer")
	}
}

func TestFindWithExtensionFilter(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.go", "b.rs", "c.go"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	f, err := New(&Config{
		Extensions:       []string{"go"},
		IgnoreHidden:     true,
		ReadFdignore:     false,
		ReadVcsignore:    false,
		RequireGit:       true,
		ReadParentIgnore: false,
		ReadGlobalIgnore: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	f.SetPatterns([]*regexp.Regexp{regexp.MustCompile(`(?is).*`)})

	results, err := f.Find(context.Background(), []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 .go files, got %d: %v", len(results), results)
	}
}
