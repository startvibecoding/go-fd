package main

import "testing"

func TestParseArgsBasic(t *testing.T) {
	opts, baseDir, err := parseArgs([]string{"-e", "go", "-H", "pattern", "src"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Pattern != "pattern" {
		t.Errorf("pattern = %q", opts.Pattern)
	}
	if len(opts.Paths) != 1 || opts.Paths[0] != "src" {
		t.Errorf("paths = %v", opts.Paths)
	}
	if !opts.Hidden {
		t.Error("expected hidden")
	}
	if len(opts.Extensions) != 1 || opts.Extensions[0] != "go" {
		t.Errorf("extensions = %v", opts.Extensions)
	}
	if baseDir != "" {
		t.Errorf("baseDir = %q", baseDir)
	}
}

func TestParseShortCluster(t *testing.T) {
	opts, _, err := parseArgs([]string{"-Lp", "foo"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.FollowLinks || !opts.FullPath {
		t.Errorf("expected follow+fullpath, got %+v", opts)
	}
}

func TestParseAttachedValue(t *testing.T) {
	opts, _, err := parseArgs([]string{"-tf", "-d3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.Types) != 1 || opts.Types[0] != "f" {
		t.Errorf("types = %v", opts.Types)
	}
	if opts.MaxDepth != 3 {
		t.Errorf("max-depth = %d", opts.MaxDepth)
	}
}

func TestParseExec(t *testing.T) {
	opts, _, err := parseArgs([]string{"pattern", "-x", "echo", "{}", ";", "extra"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.Exec) != 2 || opts.Exec[0] != "echo" || opts.Exec[1] != "{}" {
		t.Errorf("exec = %v", opts.Exec)
	}
	// "extra" after ';' is a positional path.
	if len(opts.Paths) != 1 || opts.Paths[0] != "extra" {
		t.Errorf("paths = %v", opts.Paths)
	}
}

func TestParseDoubleDash(t *testing.T) {
	opts, _, err := parseArgs([]string{"--", "-foo"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Pattern != "-foo" {
		t.Errorf("pattern = %q", opts.Pattern)
	}
}

func TestParseLongWithEquals(t *testing.T) {
	opts, _, err := parseArgs([]string{"--max-results=5", "--color=never"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.MaxResults != 5 {
		t.Errorf("max-results = %d", opts.MaxResults)
	}
	if opts.Color != "never" {
		t.Errorf("color = %q", opts.Color)
	}
}
